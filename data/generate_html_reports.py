import os
import sqlite3
import pandas as pd
import json
import math
import argparse

# ===============================
# CONFIGURATION
# ===============================
DB_PATH = "vudatasim.db"
TABLE_NAME = "test_runs"
OUTPUT_DIR = "html_reports"

# ===============================
# FIELD GROUP DEFINITIONS
# ===============================

SUMMARY_FIELDS = [
    "test_id", "target_eps", "start_time", "end_time",
    "o11y_sources", "timeout_seconds", "status"
]

POD_GROUPS = {
    "Kafka Cluster Pods": {
        "pods": ["kafka-cluster-cp-kafka-0", "kafka-cluster-cp-kafka-1", "kafka-cluster-cp-kafka-2"]
    },
    "ClickHouse Pods": {
        "pods": ["chi-clickhouse-vusmart-0-0-0", "chi-clickhouse-vusmart-0-1-0"]
    }
}


PIPELINE_GROUPS = {
    "Pipeline Pod CPU": {
        "rows": ["pipeline_pod"],
        "cols": ["cpu_min", "cpu_avg", "cpu_max"]
    },
    "Pipeline Pod Memory": {
        "rows": ["pipeline_pod"],
        "cols": ["mem_min", "mem_avg", "mem_max"]
    }
}

NODE_GROUPS = {
    "Kafka Node CPU": {
        "rows": ["kafka_1_node", "kafka_2_node", "kafka_3_node"],
        "cols": ["cpu_min", "cpu_avg", "cpu_max"]
    },
    "Kafka Node Memory": {
        "rows": ["kafka_1_node", "kafka_2_node", "kafka_3_node"],
        "cols": ["mem_min", "mem_avg", "mem_max"]
    },
    "ClickHouse Node CPU": {
        "rows": ["ch1_node", "ch2_node"],
        "cols": ["cpu_min", "cpu_avg", "cpu_max"]
    },
    "ClickHouse Node Memory": {
        "rows": ["ch1_node", "ch2_node"],
        "cols": ["mem_min", "mem_avg", "mem_max"]
    }
}

# ===============================
# NODE FIELD MAPPING
# ===============================
NODE_ALLOCATED_MAP = {
    # Kafka nodes
    "kafka_1_node_cpu_total": "kafka_1_node_cpu_total",
    "kafka_1_node_mem_total": "kafka_1_node_mem_total",
    "kafka_2_node_cpu_total": "kafka_2_node_cpu_total",
    "kafka_2_node_mem_total": "kafka_2_node_mem_total",
    "kafka_3_node_cpu_total": "kafka_3_node_cpu_total",
    "kafka_3_node_mem_total": "kafka_3_node_mem_total",
    # ClickHouse nodes
    "ch1_node_cpu_total": "ch1_node_cpu_total",
    "ch1_node_mem_total": "ch1_node_mem_total",
    "ch2_node_cpu_total": "ch2_node_cpu_total",
    "ch2_node_mem_total": "ch2_node_mem_total",
}

# ===============================
# HELPER FUNCTIONS
# ===============================

def safe_get(row, key):
    return row[key] if key in row and pd.notna(row[key]) else ""

def smart_format(value):
    """Round only if float and not an integer. Keep integers unchanged."""
    try:
        num = float(value)
        if math.isclose(num, int(num)):
            return str(int(num))
        else:
            return f"{num:.2f}"
    except Exception:
        return value

def format_duration_from_seconds(seconds):
    """Convert seconds to hours string (e.g., 1h, 1.5h)."""
    try:
        hours = float(seconds) / 3600
        if hours.is_integer():
            return f"{int(hours)}h"
        else:
            return f"{hours:.1f}h"
    except Exception:
        return "unknown"

def convert_eps_to_k(eps):
    """Convert 5000 → 5k, 125000 → 125k."""
    try:
        eps_val = float(eps)
        return f"{int(eps_val / 1000)}k"
    except Exception:
        return str(eps)

def html_section(title, content):
    return f"<section><h2>{title}</h2>{content}</section>"

def extract_pipeline_name_from_info(row):
    info_data = safe_get(row, "pipeline_info")
    if not info_data:
        return None
    try:
        info = json.loads(info_data)
        if isinstance(info, dict) and len(info) > 0:
            first_entry = next(iter(info.values()))
            return first_entry.get("name", "unknown_pipeline")
    except Exception:
        pass
    return "unknown_pipeline"

# ===============================
# HTML SECTION FORMATTERS
# ===============================

# ===============================
# NODE-POD MAPPING TABLE
# ===============================

# ===============================
# NODE-POD MAPPING TABLE (FIXED)
# ===============================

# ===============================
# NODE-POD MAPPING TABLE (ROWSPAN)
# ===============================

def format_node_pod_table(row):
    """
    Builds a Node → Pods mapping with only one row per node using HTML rowspan.
    Combines Pod CPU & MEM into a single column: 'X cores / Y Gi'.
    """
    try:
        pods_cpu    = json.loads(safe_get(row, "pods_cpu"))
        pods_memory = json.loads(safe_get(row, "pods_memory"))
        nodes_cpu   = json.loads(safe_get(row, "nodes_cpu"))
        nodes_memory = json.loads(safe_get(row, "nodes_memory"))
    except Exception as e:
        return html_section("⚠️ Node-Pod Table", f"<p>JSON Parsing Failed: {e}</p>")

    table_html = (
        "<table><thead><tr>"
        "<th>Node Name</th><th>Node Resources</th>"
        "<th>Pod</th><th>Pod Resources</th><th>Group</th>"
        "</tr></thead><tbody>"
    )

    for node_name, node_cpu_info in nodes_cpu.items():
        node_cpu_alloc = smart_format(node_cpu_info.get("allocated", ""))
        node_mem_alloc = smart_format(nodes_memory.get(node_name, {}).get("allocated", ""))

        node_pods = []
        for pod_name, cpu_info in pods_cpu.items():
            if cpu_info.get("node_name") == node_name:
                pod_cpu_alloc = smart_format(cpu_info.get("allocated", ""))
                pod_mem_alloc = smart_format(pods_memory.get(pod_name, {}).get("allocated", ""))

                pod_lower = pod_name.lower()
                if "kafka" in pod_lower:
                    group = "Kafka"
                elif "clickhouse" in pod_lower:
                    group = "CH"
                elif "traefik" in pod_lower:
                    group = "Traefik"
                else:
                    group = "Pipeline"

                node_pods.append((pod_name, pod_cpu_alloc, pod_mem_alloc, group))

        if not node_pods:
            table_html += (
                f"<tr>"
                f"<td>{node_name}</td>"
                f"<td>{node_cpu_alloc} cores / {node_mem_alloc} Gi</td>"
                f"<td colspan='3'>No pods</td>"
                f"</tr>"
            )
            continue

        rowspan = len(node_pods)
        first_pod = node_pods[0]

        table_html += (
            f"<tr>"
            f"<td rowspan='{rowspan}'>{node_name}</td>"
            f"<td rowspan='{rowspan}'>{node_cpu_alloc} cores / {node_mem_alloc} Gi</td>"
            f"<td>{first_pod[0]}</td>"
            f"<td>{first_pod[1]} cores / {first_pod[2]} Gi</td>"
            f"<td>{first_pod[3]}</td>"
            f"</tr>"
        )

        for pod in node_pods[1:]:
            table_html += (
                f"<tr>"
                f"<td>{pod[0]}</td>"
                f"<td>{pod[1]} cores / {pod[2]} Gi</td>"
                f"<td>{pod[3]}</td>"
                f"</tr>"
            )

    table_html += "</tbody></table>"
    return html_section("🧩 Node/Pod Resources", table_html)


def add_percent_to_headers(html_table):
    """
    Automatically adds '%' to column names that contain Min, Avg, or Max.
    Works on ALL tables without manual edits.
    """
    return (
        html_table
        .replace("<th>CPU Min</th>", "<th>CPU Min (%)</th>")
        .replace("<th>CPU Avg</th>", "<th>CPU Avg (%)</th>")
        .replace("<th>CPU Max</th>", "<th>CPU Max (%)</th>")
        .replace("<th>MEM Min</th>", "<th>MEM Min (%)</th>")
        .replace("<th>MEM Avg</th>", "<th>MEM Avg (%)</th>")
        .replace("<th>MEM Max</th>", "<th>MEM Max (%)</th>")
        .replace("<th>Min</th>", "<th>Min (%)</th>")
        .replace("<th>Avg</th>", "<th>Avg (%)</th>")
        .replace("<th>Max</th>", "<th>Max (%)</th>")
    )

def get_pod_restarts(pod_name, pod_restarts_dict):
    """
    Fetch restart value(s) from pod_restarts JSON.
    If multiple matches (for pipeline prefixes), sum them.
    Example: 'pipeline-prod' → pipeline-prod-0, pipeline-prod-1 ...
    """
    restarts = 0
    for key, val in pod_restarts_dict.items():
        if key.startswith(pod_name):  # prefix match
            try:
                restarts += int(val)
            except:
                pass
    return restarts


def format_node_tables(row):
    """
    Builds separate Kafka & ClickHouse node tables.
    Each row: pod_name | node_name | CPU_ALLOC | CPU_MIN | CPU_AVG | CPU_MAX | MEM_ALLOC | MEM_MIN | MEM_AVG | MEM_MAX
    """
    try:
        pods_cpu    = json.loads(safe_get(row, "pods_cpu"))
        pods_memory = json.loads(safe_get(row, "pods_memory"))
        nodes_cpu   = json.loads(safe_get(row, "nodes_cpu"))
        nodes_memory = json.loads(safe_get(row, "nodes_memory"))
    except Exception as e:
        return html_section("⚠️ Node Tables", f"<p>JSON Parsing Failed: {e}</p>")

    kafka_rows = []
    clickhouse_rows = []

    for pod_name, cpu_info in pods_cpu.items():
        node = cpu_info.get("node_name")
        if not node:
            continue  

        cpu_stats = nodes_cpu.get(node, {})
        mem_stats = nodes_memory.get(node, {})

        row_data = {
            "pod": pod_name,
            "node": node,
            "cpu_alloc": smart_format(cpu_stats.get("allocated", "")),
            "cpu_min": smart_format(cpu_stats.get("min", "")),
            "cpu_avg": smart_format(cpu_stats.get("avg", "")),
            "cpu_max": smart_format(cpu_stats.get("max", "")),
            "mem_alloc": smart_format(mem_stats.get("allocated", "")),
            "mem_min": smart_format(mem_stats.get("min", "")),
            "mem_avg": smart_format(mem_stats.get("avg", "")),
            "mem_max": smart_format(mem_stats.get("max", "")),
        }

        if "kafka" in pod_name.lower():
            kafka_rows.append(row_data)
        elif "clickhouse" in pod_name.lower():
            clickhouse_rows.append(row_data)

    def build_table(title, rows):
        if not rows:
            return html_section(title, "<p>No data found.</p>")

        html = (
            f"<table><thead><tr>"
            f"<th>Pod Name</th><th>Node Name</th>"
            f"<th>CPU Alloc</th><th>CPU Min (%)</th><th>CPU Avg (%)</th><th>CPU Max (%)</th>"
            f"<th>MEM Alloc</th><th>MEM Min (%)</th><th>MEM Avg (%)</th><th>MEM Max (%)</th>"
            f"</tr></thead><tbody>"
        )

        # one row per node – using the first pod for each
        seen_nodes = set()
        for r in rows:
            if r["node"] in seen_nodes:
                continue
            seen_nodes.add(r["node"])

            html += (
                f"<tr>"
                f"<td>{r['pod']}</td>"
                f"<td>{r['node']}</td>"
                f"<td>{r['cpu_alloc']} cores</td><td>{r['cpu_min']}</td>"
                f"<td>{r['cpu_avg']}</td><td>{r['cpu_max']}</td>"
                f"<td>{r['mem_alloc']} Gi</td><td>{r['mem_min']}</td>"
                f"<td>{r['mem_avg']}</td><td>{r['mem_max']}</td>"
                f"</tr>"
            )

        return html_section(title, html + "</tbody></table>")

    # Build separate tables
    html_kafka = build_table("🐳 Kafka Node CPU + Memory", kafka_rows)
    html_clickhouse = build_table("🏛️ ClickHouse Node CPU + Memory", clickhouse_rows)

    return html_kafka + html_clickhouse

def format_process_rate_table(row):
    data = safe_get(row, "process_rate_summary")
    if not data:
        return html_section("⚙️ Process Rate Metrics", "<p>⚠️ No process rate data found.</p>")

    try:
        pr = json.loads(data)
    except Exception:
        return html_section("⚙️ Process Rate Metrics", "<p>⚠️ JSON parsing failed.</p>")

    if len(pr.keys()) == 1:
        # Single source - retain existing table
        key = list(pr.keys())[0]
        values = pr[key]

        min_rate = smart_format(values.get("min_process_rate", ""))
        avg_rate = smart_format(values.get("avg_process_rate", ""))
        max_rate = smart_format(values.get("max_process_rate", ""))

        html = (
            "<table><thead>"
            "<tr><th>Type</th><th>Min (msg/s)</th><th>Avg (msg/s)</th><th>Max (msg/s)</th></tr>"
            "</thead><tbody>"
            f"<tr><td>Process Rate</td><td>{min_rate}</td><td>{avg_rate}</td><td>{max_rate}</td></tr>"
            "</tbody></table>"
        )
    else:
        # Multiple sources - table with Source column
        html = (
            "<table><thead>"
            "<tr><th>Source</th><th>Min (msg/s)</th><th>Avg (msg/s)</th><th>Max (msg/s)</th></tr>"
            "</thead><tbody>"
        )
        for key, values in pr.items():
            min_rate = smart_format(values.get("min_process_rate", ""))
            avg_rate = smart_format(values.get("avg_process_rate", ""))
            max_rate = smart_format(values.get("max_process_rate", ""))

            html += f"<tr><td>{key}</td><td>{min_rate}</td><td>{avg_rate}</td><td>{max_rate}</td></tr>"
        html += "</tbody></table>"

    return html_section("⚙️ Process Rate Metrics", html)



def format_summary(row):
    html = "<ul>"
    for field in SUMMARY_FIELDS:
        value = smart_format(safe_get(row, field))
        label = field.replace("_", " ").title()
        if field == "timeout_seconds":
            try:
                hours = float(value) / 3600
                value = f"{hours:.2f} hours"
            except Exception:
                pass
            label = "Duration"
        html += f"<li><strong>{label}</strong>: <code>{value}</code></li>"

    # Check if traces is the o11y source
    try:
        o11y_sources = json.loads(safe_get(row, "o11y_sources")) or []
    except:
        o11y_sources = []
    is_traces = "traces" in o11y_sources

    if is_traces:
        # For traces, get observed avg input from topic_metrics_json
        topic_metrics_data = safe_get(row, "topic_metrics_json")
        avg_in = "N/A"
        if topic_metrics_data:
            try:
                metrics = json.loads(topic_metrics_data)
                traces_data = metrics.get("traces", {})
                input_avgs = []
                for topic, data in traces_data.items():
                    if data.get("topic_type") == "input" and data.get("avg"):
                        input_avgs.append(data["avg"])
                if input_avgs:
                    avg_in = smart_format(sum(input_avgs) / len(input_avgs))
            except:
                pass
        html += f"<li><strong>Observed Avg Input Topic EPS</strong>: <code>{avg_in}</code></li>"
    else:
        avg_in = smart_format(safe_get(row, "avg_input_msgs_per_sec"))
        avg_out = smart_format(safe_get(row, "avg_output_msgs_per_sec"))
        html += f"<li><strong>Observed Avg Input Topic EPS</strong>: <code>{avg_in}</code></li>"
        html += f"<li><strong>Observed Avg Output Topic EPS</strong>: <code>{avg_out}</code></li>"

    html += "</ul>"
    return html_section("🧾 Test Summary", html)

def format_pipeline_info(row):
    info_data = safe_get(row, "pipeline_info")
    if not info_data:
        return "", None  # return pipeline table + pipeline name as None

    try:
        info = json.loads(info_data)
        html = "<table><thead><tr><th>Source</th><th>Pipeline Name</th><th>Threads</th><th>Instances</th></tr></thead><tbody>"

        pipeline_name = None
        for source, details in info.items():
            pipeline_name = details.get('name', '')
            html += (
                f"<tr><td>{source}</td><td>{pipeline_name}</td>"
                f"<td>{smart_format(details.get('threads',''))}</td>"
                f"<td>{smart_format(details.get('instances',''))}</td></tr>"
            )

        html += "</tbody></table>"
        return html_section("🔧 Pipeline Info", html), pipeline_name

    except Exception:
        return html_section("🔧 Pipeline Info", f"<p>⚠️ Could not parse JSON: {info_data}</p>"), None

def format_pipeline_pods_from_json(title, pipeline_info_data, row):
    if not pipeline_info_data:
        return ""  # No pipeline info found

    try:
        pods_cpu = json.loads(safe_get(row, "pods_cpu"))
        pods_mem = json.loads(safe_get(row, "pods_memory"))
    except Exception:
        return html_section(title, "<p>⚠️ Pods JSON parsing failed.</p>")
    try:
        pod_restarts = json.loads(safe_get(row, "pod_restarts"))
    except Exception:
        pod_restarts = {}   # fallback

    # Check if pipeline_info_data is a string (for traces prefix)
    if isinstance(pipeline_info_data, str):
        # For traces, use the string as prefix
        pipeline_name = pipeline_info_data
        matched_pods = [
            pod_name for pod_name in pods_cpu.keys()
            if pod_name.startswith(pipeline_name)
        ]
        all_matched_pods = [(pipeline_name, pod) for pod in matched_pods]
    else:
        try:
            pipeline_info = json.loads(pipeline_info_data)
        except Exception:
            return html_section(title, "<p>⚠️ Pipeline info JSON parsing failed.</p>")

        # Extract all pipeline names
        all_pipeline_names = [details.get('name', '') for details in pipeline_info.values() if details.get('name')]

        if not all_pipeline_names:
            return html_section(title, "<p>⚠️ No pipeline names found.</p>")

        # Collect all matched pods with their pipeline
        all_matched_pods = []
        for pipeline_name in all_pipeline_names:
            matched_pods = [
                pod_name for pod_name in pods_cpu.keys()
                if pod_name.startswith(pipeline_name)
            ]
            for pod in matched_pods:
                all_matched_pods.append((pipeline_name, pod))

    if not all_matched_pods:
        return html_section(title, "<p>⚠️ No matching pipeline pods found.</p>")

    # Build table with Pipeline column
    html = (
        f"<table><thead><tr>"
        f"<th>Pipeline</th><th>Pod</th><th>CPU Allocated</th><th>CPU Min</th><th>CPU Avg</th><th>CPU Max</th>"
        f"<th>MEM Allocated</th><th>MEM Min</th><th>MEM Avg</th><th>MEM Max</th><th>Restarts</th>"
        f"</tr></thead><tbody>"
    )

    for pipeline_name, pod in all_matched_pods:
        cpu = pods_cpu.get(pod, {})
        mem = pods_mem.get(pod, {})

        restarts = get_pod_restarts(pod, pod_restarts)

        html += (
            f"<tr>"
            f"<td>{pipeline_name}</td>"
            f"<td>{pod}</td>"
            f"<td>{smart_format(cpu.get('allocated', ''))} cores</td>"
            f"<td>{smart_format(cpu.get('min', ''))}</td>"
            f"<td>{smart_format(cpu.get('avg', ''))}</td>"
            f"<td>{smart_format(cpu.get('max', ''))}</td>"
            f"<td>{smart_format(mem.get('allocated', ''))} Gi</td>"
            f"<td>{smart_format(mem.get('min', ''))}</td>"
            f"<td>{smart_format(mem.get('avg', ''))}</td>"
            f"<td>{smart_format(mem.get('max', ''))}</td>"
            f"<td>{restarts}</td>"
            f"</tr>"
        )

    return html_section(title, html + "</tbody></table>")


def format_pod_group_from_json(title, pod_list, row):
    """
    Fetches allocated / min / avg / max values for Kafka & ClickHouse pods 
    **ONLY from `pods_cpu` and `pods_memory` JSON**.
    """
    try:
        pods_cpu = json.loads(safe_get(row, "pods_cpu"))
        pods_mem = json.loads(safe_get(row, "pods_memory"))
    except Exception:
        return html_section(title, "<p>⚠️ JSON parsing failed.</p>")
    try:
        pod_restarts = json.loads(safe_get(row, "pod_restarts"))
    except Exception:
        pod_restarts = {}   # fallback


    html = (
        f"<table><thead><tr>"
        f"<th>Pod</th><th>CPU Allocated</th><th>CPU Min</th><th>CPU Avg</th><th>CPU Max</th>"
        f"<th>MEM Allocated</th><th>MEM Min</th><th>MEM Avg</th><th>MEM Max</th><th>Restarts</th>"
        f"</tr></thead><tbody>"
    )

    for pod in pod_list:
        cpu_data = pods_cpu.get(pod, {})
        mem_data = pods_mem.get(pod, {})

        restarts = get_pod_restarts(pod, pod_restarts)

        html += (
            f"<tr>"
            f"<td>{pod}</td>"
            f"<td>{smart_format(cpu_data.get('allocated', ''))} cores</td>"
            f"<td>{smart_format(cpu_data.get('min', ''))}</td>"
            f"<td>{smart_format(cpu_data.get('avg', ''))}</td>"
            f"<td>{smart_format(cpu_data.get('max', ''))}</td>"
            f"<td>{smart_format(mem_data.get('allocated', ''))} Gi</td>"
            f"<td>{smart_format(mem_data.get('min', ''))}</td>"
            f"<td>{smart_format(mem_data.get('avg', ''))}</td>"
            f"<td>{smart_format(mem_data.get('max', ''))}</td>"
            f"<td>{restarts}</td>"
            f"</tr>"
        )


    html += "</tbody></table>"
    return html_section(title, html)


def format_topic_metrics_tables(row):
    topic_metrics_data = safe_get(row, "topic_metrics_json")
    try:
        o11y_sources = json.loads(safe_get(row, "o11y_sources")) or []
    except:
        o11y_sources = []
    is_traces = "traces" in o11y_sources

    if is_traces:
        # For traces, only show input topics from topic_metrics_json
        html = "<section><h2>📈 Input/Output Topic Metrics</h2>"
        if topic_metrics_data:
            try:
                metrics = json.loads(topic_metrics_data)
                traces_data = metrics.get("traces", {})
                input_metrics = [data for data in traces_data.values() if data.get("topic_type") == "input"]
                html += generate_source_input_table(input_metrics)
            except Exception as e:
                html += f"<p>⚠️ Error parsing topic metrics: {e}</p>"
        html += "</section>"
        return html
    elif not topic_metrics_data or len(o11y_sources) <= 1:
        # Single source or no data - use existing combined table
        html = (
            "<table><thead><tr><th>Type</th><th>Min (msg/s)</th><th>Avg (msg/s)</th><th>Max (msg/s)</th></tr></thead><tbody>"
            f"<tr><td>Input</td><td>{smart_format(safe_get(row,'min_input_msgs_per_sec'))}</td>"
            f"<td>{smart_format(safe_get(row,'avg_input_msgs_per_sec'))}</td>"
            f"<td>{smart_format(safe_get(row,'max_input_msgs_per_sec'))}</td></tr>"
            f"<tr><td>Output</td><td>{smart_format(safe_get(row,'min_output_msgs_per_sec'))}</td>"
            f"<td>{smart_format(safe_get(row,'avg_output_msgs_per_sec'))}</td>"
            f"<td>{smart_format(safe_get(row,'max_output_msgs_per_sec'))}</td></tr>"
            "</tbody></table>"
        )
        return html_section("📈 Input/Output Topic Metrics", html)
    else:
        # Multiple sources - separate tables per source
        html = "<section><h2>📈 Input/Output Topic Metrics</h2>"

        try:
            metrics = json.loads(topic_metrics_data)

            for source in o11y_sources:
                source_data = metrics.get(source, {})
                if not source_data:
                    continue

                html += f"<h3>{source} Topic Metrics</h3>"

                # Separate input metrics and output topic metrics
                input_metrics = []
                output_topic_metrics = {}

                for topic_name, topic_data in source_data.items():
                    if topic_data.get("topic_type") == "input":
                        input_metrics.append(topic_data)
                    elif topic_data.get("topic_type") == "output":
                        output_topic_metrics[topic_name] = topic_data

                # Generate separate tables for this source
                html += generate_source_input_table(input_metrics)
                html += generate_source_output_table(output_topic_metrics)

        except Exception as e:
            html += f"<p>⚠️ Error parsing topic metrics: {e}</p>"

        html += "</section>"
        return html

def generate_source_input_table(input_metrics):
    """Generate separate table for input topic metrics (aggregated)"""

    def aggregate_metrics(topic_list):
        if not topic_list:
            return {"min": "", "avg": "", "max": ""}

        mins = [t.get("min", 0) for t in topic_list if t.get("min") is not None]
        avgs = [t.get("avg", 0) for t in topic_list if t.get("avg") is not None]
        maxs = [t.get("max", 0) for t in topic_list if t.get("max") is not None]

        return {
            "min": smart_format(min(mins)) if mins else "",
            "avg": smart_format(sum(avgs)/len(avgs)) if avgs else "",
            "max": smart_format(max(maxs)) if maxs else ""
        }

    # Aggregate input metrics
    input_agg = aggregate_metrics(input_metrics) if input_metrics else {"min": "", "avg": "", "max": ""}

    table_html = f"""
    <h4>Input Topic Message Rates</h4>
    <table>
        <thead><tr><th>Topic/Type</th><th>Min (msg/s)</th><th>Avg (msg/s)</th><th>Max (msg/s)</th></tr></thead>
        <tbody>
            <tr><td>Input</td><td>{input_agg['min']}</td><td>{input_agg['avg']}</td><td>{input_agg['max']}</td></tr>
        </tbody>
    </table>
    """
    return table_html

def generate_source_output_table(output_topic_metrics):
    """Generate separate table for output topic metrics (individual + total)"""

    if not output_topic_metrics:
        return ""

    table_html = f"""
    <h4>Output Topic Message Rates</h4>
    <table>
        <thead><tr><th>Topic</th><th>Min (msg/s)</th><th>Avg (msg/s)</th><th>Max (msg/s)</th></tr></thead>
        <tbody>
    """

    total_min = 0
    total_avg = 0
    total_max = 0
    count = 0

    # Add individual output topic rows and accumulate totals
    for topic_name, metrics in output_topic_metrics.items():
        min_val = metrics.get('min', 0) or 0
        avg_val = metrics.get('avg', 0) or 0
        max_val = metrics.get('max', 0) or 0

        total_min += min_val
        total_avg += avg_val
        total_max += max_val
        count += 1

        table_html += f"<tr><td>{topic_name}</td><td>{smart_format(min_val)}</td><td>{smart_format(avg_val)}</td><td>{smart_format(max_val)}</td></tr>"

    # Add total row
    if count > 1:
        table_html += f"<tr style='font-weight: bold; border-top: 2px solid #333;'><td>Total</td><td>{smart_format(total_min)}</td><td>{smart_format(total_avg)}</td><td>{smart_format(total_max)}</td></tr>"

    table_html += "</tbody></table>"
    return table_html

def format_lag_table(row):
    o11y_sources = []
    try:
        o11y_sources = json.loads(safe_get(row, "o11y_sources")) or []
    except Exception:
        pass

    lag_data = safe_get(row, "lag_per_source")
    if not lag_data:
        # Fallback to original logic if no lag_per_source
        html = (
            "<p><strong>Note:</strong> Lag is in terms of offsets with respect Kafka and ContextStream Pipeline.</p>"
            "<table><thead><tr><th>Min Lag</th><th>Avg Lag</th><th>Max Lag</th></tr></thead><tbody>"
            f"<tr><td>{smart_format(safe_get(row,'min_lag'))}</td>"
            f"<td>{smart_format(safe_get(row,'avg_lag'))}</td>"
            f"<td>{smart_format(safe_get(row,'max_lag'))}</td></tr>"
            "</tbody></table>"
        )
        return html_section("⏱️ Lag Metrics", html)

    try:
        lag_per_source = json.loads(lag_data)
    except Exception:
        return html_section("⏱️ Lag Metrics", "<p>⚠️ JSON parsing failed for lag data.</p>")

    html = "<p><strong>Note:</strong> Lag is in terms of offsets with respect Kafka and ContextStream Pipeline.</p>"

    if len(o11y_sources) == 1:
        # Single source: Retain same table structure (no O11y Source column)
        source = o11y_sources[0]
        metrics = lag_per_source.get(source, {})
        min_lag = smart_format(metrics.get("min", ""))
        avg_lag = smart_format(metrics.get("avg", ""))
        max_lag = smart_format(metrics.get("max", ""))

        html += (
            "<table><thead><tr><th>Min Lag</th><th>Avg Lag</th><th>Max Lag</th></tr></thead><tbody>"
            f"<tr><td>{min_lag}</td><td>{avg_lag}</td><td>{max_lag}</td></tr>"
            "</tbody></table>"
        )
    else:
        # Multiple sources: Single table with O11y Source column, rows for each source
        html += (
            "<table><thead><tr><th>O11y Source</th><th>Min Lag</th><th>Avg Lag</th><th>Max Lag</th></tr></thead><tbody>"
        )

        for source, metrics in lag_per_source.items():
            min_lag = smart_format(metrics.get("min", ""))
            avg_lag = smart_format(metrics.get("avg", ""))
            max_lag = smart_format(metrics.get("max", ""))

            html += f"<tr><td>{source}</td><td>{min_lag}</td><td>{avg_lag}</td><td>{max_lag}</td></tr>"

        html += "</tbody></table>"

    return html_section("⏱️ Lag Metrics", html)


def format_grouped_table(title, group_def, row):
    """Handles Pods, Pipelines, and Nodes with correct allocated logic."""
    is_cpu = "CPU" in title
    is_memory = "Memory" in title
    unit = "Cores" if is_cpu else "Gi"
    allocated_label = f"Allocated ({unit})"
    is_pipeline_group = "Pipeline Pod" in title
    is_node_group = "Node" in title

    pipeline_name = extract_pipeline_name_from_info(row) if is_pipeline_group else None

    html = (
        f"<table><thead><tr><th>{'Pipeline Name' if is_pipeline_group else 'Component'}</th>"
        f"<th>{allocated_label}</th><th>Min (%)</th><th>Avg (%)</th><th>Max (%)</th></tr></thead><tbody>"
    )

    for comp in group_def["rows"]:
        # Determine allocated field logic
        if is_node_group:
            alloc_field = f"{comp}_{'cpu_total' if is_cpu else 'mem_total'}"
        else:
            alloc_field = f"{comp}_{'cpu' if is_cpu else 'mem'}_allocated"

        min_field, avg_field, max_field = [f"{comp}_{col}" for col in group_def["cols"]]

        html += (
            f"<tr><td>{pipeline_name if is_pipeline_group else comp}</td>"
            f"<td>{smart_format(safe_get(row, alloc_field))}</td>"
            f"<td>{smart_format(safe_get(row, min_field))}</td>"
            f"<td>{smart_format(safe_get(row, avg_field))}</td>"
            f"<td>{smart_format(safe_get(row, max_field))}</td></tr>"
        )

    html += "</tbody></table>"
    return html_section(title, html)

def format_clickhouse_ingestion_table(row):
    """
    Parses the `ingestion_summary` column and creates tables:
    - Single source: One table with Table | Min EPS | Avg EPS | Max EPS
    - Multiple sources: Separate tables per o11y_source
    """
    data = safe_get(row, "ingestion_summary")
    if not data:
        return html_section("🏭 ClickHouse Ingestion Summary", "<p>⚠️ No ingestion data found.</p>")

    try:
        ingestion = json.loads(data)
    except Exception:
        return html_section("🏭 ClickHouse Ingestion Summary", "<p>⚠️ JSON parsing failed.</p>")

    # Normalize to list of dicts
    if isinstance(ingestion, dict):
        ingestion = [ingestion]

    if not isinstance(ingestion, list):
        return html_section("🏭 ClickHouse Ingestion Summary", "<p>⚠️ Unexpected data format.</p>")

    # Collect unique o11y sources
    unique_sources = set()
    for item in ingestion:
        source = item.get("o11y_source", "unknown")
        unique_sources.add(source)

    if len(unique_sources) <= 1:
        # Single source - use existing table format
        html = (
            "<table><thead><tr>"
            "<th>Table</th><th>Min EPS</th><th>Avg EPS</th><th>Max EPS</th>"
            "</tr></thead><tbody>"
        )

        total_min = 0
        total_avg = 0
        total_max = 0

        for item in ingestion:
            table = item.get("table", "unknown")
            min_eps = smart_format(item.get("min_eps", ""))
            avg_eps = smart_format(item.get("avg_eps", ""))
            max_eps = smart_format(item.get("max_eps", ""))

            # Accumulate totals
            try:
                total_min += float(item.get("min_eps", 0) or 0)
            except:
                pass
            try:
                total_avg += float(item.get("avg_eps", 0) or 0)
            except:
                pass
            try:
                total_max += float(item.get("max_eps", 0) or 0)
            except:
                pass

            html += (
                f"<tr>"
                f"<td>{table}</td>"
                f"<td>{min_eps}</td>"
                f"<td>{avg_eps}</td>"
                f"<td>{max_eps}</td>"
                f"</tr>"
            )

        # Add total row
        html += (
            f"<tr style='font-weight: bold; border-top: 2px solid #333;'>"
            f"<td>Total</td>"
            f"<td>{smart_format(total_min)}</td>"
            f"<td>{smart_format(total_avg)}</td>"
            f"<td>{smart_format(total_max)}</td>"
            f"</tr>"
        )

        html += "</tbody></table>"
        return html_section("🏭 ClickHouse Ingestion Summary", html)
    else:
        # Multiple sources - separate tables per source
        html = "<section><h2>🏭 ClickHouse Ingestion Summary</h2>"

        # Group data by source
        source_data = {}
        for item in ingestion:
            source = item.get("o11y_source", "unknown")
            if source not in source_data:
                source_data[source] = []
            source_data[source].append(item)

        # Create table for each source
        for source, items in source_data.items():
            html += f"<h3>{source}</h3>"
            html += (
                "<table><thead><tr>"
                "<th>Table</th><th>Min EPS</th><th>Avg EPS</th><th>Max EPS</th>"
                "</tr></thead><tbody>"
            )

            total_min = 0
            total_avg = 0
            total_max = 0

            for item in items:
                table = item.get("table", "unknown")
                min_eps = smart_format(item.get("min_eps", ""))
                avg_eps = smart_format(item.get("avg_eps", ""))
                max_eps = smart_format(item.get("max_eps", ""))

                # Accumulate totals
                try:
                    total_min += float(item.get("min_eps", 0) or 0)
                except:
                    pass
                try:
                    total_avg += float(item.get("avg_eps", 0) or 0)
                except:
                    pass
                try:
                    total_max += float(item.get("max_eps", 0) or 0)
                except:
                    pass

                html += (
                    f"<tr>"
                    f"<td>{table}</td>"
                    f"<td>{min_eps}</td>"
                    f"<td>{avg_eps}</td>"
                    f"<td>{max_eps}</td>"
                    f"</tr>"
                )

            # Add total row
            html += (
                f"<tr style='font-weight: bold; border-top: 2px solid #333;'>"
                f"<td>Total</td>"
                f"<td>{smart_format(total_min)}</td>"
                f"<td>{smart_format(total_avg)}</td>"
                f"<td>{smart_format(total_max)}</td>"
                f"</tr>"
            )

            html += "</tbody></table>"

        html += "</section>"
        return html


def get_topics_by_source(topic_metrics_json):
    """Extract topic names grouped by o11y source from topic_metrics_json"""
    try:
        metrics = json.loads(topic_metrics_json)
        topics_by_source = {}
        for source, topics in metrics.items():
            topics_by_source[source] = list(topics.keys())
        return topics_by_source
    except:
        return {}

def generate_combined_topic_table(input_topics, output_topics):
    """Generate combined table for both input and output topics with Topic Type column"""
    all_topics = []

    # Add input topics
    for topic in input_topics:
        all_topics.append({
            'name': topic.get('name', ''),
            'type': 'input',
            'partitions': topic.get('partitions', ''),
            'replication': topic.get('replication_factor', '')
        })

    # Add output topics
    for topic in output_topics:
        all_topics.append({
            'name': topic.get('name', ''),
            'type': 'output',
            'partitions': topic.get('partitions', ''),
            'replication': topic.get('replication_factor', '')
        })

    if not all_topics:
        return ""

    html = "<table><thead><tr><th>Topic Name</th><th>Topic Type</th><th>Partitions</th><th>Replication Factor</th></tr></thead><tbody>"
    for topic in all_topics:
        html += f"<tr><td>{topic['name']}</td><td>{topic['type']}</td><td>{smart_format(topic['partitions'])}</td><td>{smart_format(topic['replication'])}</td></tr>"
    html += "</tbody></table>"
    return html

def format_kafka_specs(row):
    """Parse and format kafka_specs JSON into separate Input and Output topic tables (HTML)."""
    specs_data = safe_get(row, "kafka_specs")
    topic_metrics_data = safe_get(row, "topic_metrics_json")

    if not specs_data:
        return ""

    html = "<section><h2>🪣 Kafka Specs</h2>"
    try:
        specs = json.loads(specs_data)
        o11y_sources = json.loads(safe_get(row, "o11y_sources"))

        if len(o11y_sources) <= 1:
            # Single source - use existing logic
            # --- Input Topics ---
            if "input_topics" in specs and isinstance(specs["input_topics"], list):
                html += (
                    "<h3>📥 Input Topics</h3>"
                    "<table><thead><tr><th>Topic Name</th><th>Partitions</th><th>Replication Factor</th></tr></thead><tbody>"
                )
                for topic in specs["input_topics"]:
                    name = topic.get("name", "")
                    partitions = topic.get("partitions", "")
                    replication = topic.get("replication_factor", "")
                    html += (
                        f"<tr><td>{name}</td>"
                        f"<td>{smart_format(partitions)}</td>"
                        f"<td>{smart_format(replication)}</td></tr>"
                    )
                html += "</tbody></table>"

            # --- Output Topics ---
            if "output_topics" in specs and isinstance(specs["output_topics"], list):
                html += (
                    "<h3>📤 Output Topics</h3>"
                    "<table><thead><tr><th>Topic Name</th><th>Partitions</th><th>Replication Factor</th></tr></thead><tbody>"
                )
                for topic in specs["output_topics"]:
                    name = topic.get("name", "")
                    partitions = topic.get("partitions", "")
                    replication = topic.get("replication_factor", "")
                    html += (
                        f"<tr><td>{name}</td>"
                        f"<td>{smart_format(partitions)}</td>"
                        f"<td>{smart_format(replication)}</td></tr>"
                    )
                html += "</tbody></table>"

            html += "</section>"
            return html
        else:
            # Multiple sources - separate tables per source
            topics_by_source = get_topics_by_source(topic_metrics_data)

            for source in o11y_sources:
                html += f"<h3>{source}</h3>"
                source_topics = topics_by_source.get(source, [])

                # Filter input topics for this source
                input_topics = [
                    t for t in specs.get("input_topics", [])
                    if t.get("name") in source_topics
                ]

                # Filter output topics for this source
                output_topics = [
                    t for t in specs.get("output_topics", [])
                    if t.get("name") in source_topics
                ]

                # Generate combined table
                html += generate_combined_topic_table(input_topics, output_topics)

            html += "</section>"
            return html

    except Exception:
        return html_section("🪣 Kafka Specs", f"<p>⚠️ Could not parse JSON: {specs_data}</p>")


def get_o11y_source_name(o11y_sources):
    try:
        src = json.loads(o11y_sources)
        if isinstance(src, list) and len(src) > 0:
            return src[0].replace(" ", "_")
    except Exception:
        pass
    return "unknown_source"

def wrap_html_template(content):
    return f"""
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Test Report</title>
<style>
body {{
font-family: 'Segoe UI', Arial, sans-serif;
margin: 2rem auto;
max-width: 1100px;
background: #f8fafc;
color: #333;
line-height: 1.6;
}}
h1, h2, h3, h4 {{
color: #1a365d;
}}
section {{
background: #fff;
padding: 1rem 1.5rem;
border-radius: 10px;
box-shadow: 0 1px 3px rgba(0,0,0,0.1);
margin-bottom: 1.5rem;
}}
table {{
width: 100%;
border-collapse: collapse;
margin-top: 1rem;
}}
th, td {{
border: 1px solid #ddd;
padding: 8px;
text-align: left;
}}
th {{
background: #1a365d;
color: white;
}}
tr:nth-child(even) {{
background-color: #f2f2f2;
}}
tr:hover {{
background-color: #e6f0ff;
}}
ul {{
list-style-type: none;
padding-left: 0;
}}
li {{
margin-bottom: 5px;
}}
code {{
background-color: #edf2f7;
padding: 2px 6px;
border-radius: 4px;
}}
</style>
</head>
<body>
{content}
</body>
</html>
"""

def generate_html_report(row):
    try:
        o11y_sources = json.loads(safe_get(row, "o11y_sources")) or []
    except:
        o11y_sources = []
    is_traces = "traces" in o11y_sources

    html_content = (
        format_summary(row)
        + format_node_pod_table(row)
        + format_kafka_specs(row)
        + format_topic_metrics_tables(row)
    )

    if not is_traces:
        html_content += format_process_rate_table(row)
        html_content += format_lag_table(row)

    html_content += format_clickhouse_ingestion_table(row)

    html_content += "<h2>🖥️ Pod Metrics</h2>"
    html_content += format_pod_group_from_json(
        "Kafka Cluster Pods",
        ["kafka-cluster-cp-kafka-0", "kafka-cluster-cp-kafka-1", "kafka-cluster-cp-kafka-2"],
        row
    )
    html_content += format_pod_group_from_json(
        "ClickHouse Pods",
        ["chi-clickhouse-vusmart-0-0-0", "chi-clickhouse-vusmart-0-1-0"],
        row
    )

    if is_traces:
        html_content += format_pipeline_pods_from_json(
            "Otel Collector", "traces-1-1-1-1", row
        )
    else:
        pipeline_info_data = safe_get(row, "pipeline_info")
        pipeline_table, _ = format_pipeline_info(row)
        html_content += pipeline_table
        html_content += format_pipeline_pods_from_json(
            "Pipeline Pods", pipeline_info_data, row
        )

    html_content += "<h2>💻 Node Metrics</h2>"
    html_content += format_node_tables(row)

    # 👉 APPLY CHANGES HERE – AUTOMATICALLY
    html_content = add_percent_to_headers(html_content)

    return wrap_html_template(html_content)




def main():
    parser = argparse.ArgumentParser(description="Generate HTML reports.")
    parser.add_argument("--test-id", type=str, help="Specific test_id to generate report for (optional)")
    args = parser.parse_args()

    os.makedirs(OUTPUT_DIR, exist_ok=True)
    conn = sqlite3.connect(DB_PATH)

    if args.test_id:
        df = pd.read_sql_query(f"SELECT * FROM {TABLE_NAME} WHERE test_id = ?", conn, params=(args.test_id,))
    else:
        df = pd.read_sql_query(f"SELECT * FROM {TABLE_NAME} WHERE report_generated = 0", conn)

    conn.close()
    print(f"✅ Loaded {len(df)} rows from database.\n")

    for _, row in df.iterrows():
        o11y_source = get_o11y_source_name(str(row["o11y_sources"]))
        eps_k = convert_eps_to_k(safe_get(row, "target_eps"))
        duration_h = format_duration_from_seconds(safe_get(row, "timeout_seconds"))
        file_name = f"{o11y_source}_{eps_k}_{duration_h}.html"
        file_path = os.path.join(OUTPUT_DIR, file_name)
        html_content = generate_html_report(row)

        with open(file_path, "w", encoding="utf-8") as f:
            f.write(html_content)
        print(f"📝 Generated: {file_path}")

        # Update report_generated to 1
        update_conn = sqlite3.connect(DB_PATH)
        update_conn.execute("UPDATE test_runs SET report_generated = 1 WHERE test_id = ?", (row['test_id'],))
        update_conn.commit()
        update_conn.close()

    print("\n✅ All HTML reports generated successfully in 'html_reports' folder.")

if __name__ == "__main__":
    main()

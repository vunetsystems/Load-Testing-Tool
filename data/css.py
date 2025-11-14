import os
import sqlite3
import pandas as pd
import json
import math

# ===============================
# CONFIGURATION
# ===============================
DB_PATH = "vudatasim.db"
TABLE_NAME = "test_runs"
OUTPUT_DIR = "css_reports"

# ===============================
# FIELD GROUP DEFINITIONS
# ===============================

SUMMARY_FIELDS = [
    "test_id", "target_eps", "start_time", "end_time",
    "o11y_sources", "timeout_seconds", "status"
]

POD_GROUPS = {
    "Kafka Cluster Pods CPU": {
        "rows": ["kafka_cluster_cp_kafka_0", "kafka_cluster_cp_kafka_1", "kafka_cluster_cp_kafka_2"],
        "cols": ["cpu_min", "cpu_avg", "cpu_max"]
    },
    "Kafka Cluster Pods Memory": {
        "rows": ["kafka_cluster_cp_kafka_0", "kafka_cluster_cp_kafka_1", "kafka_cluster_cp_kafka_2"],
        "cols": ["mem_min", "mem_avg", "mem_max"]
    },
    "ClickHouse Pods CPU": {
        "rows": ["chi_clickhouse_vusmart_0_0_0", "chi_clickhouse_vusmart_0_1_0"],
        "cols": ["cpu_min", "cpu_avg", "cpu_max"]
    },
    "ClickHouse Pods Memory": {
        "rows": ["chi_clickhouse_vusmart_0_0_0", "chi_clickhouse_vusmart_0_1_0"],
        "cols": ["mem_min", "mem_avg", "mem_max"]
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
    html += "</ul>"
    return html_section("🧾 Test Summary", html)

def format_pipeline_info(row):
    info_data = safe_get(row, "pipeline_info")
    if not info_data:
        return ""
    try:
        info = json.loads(info_data)
        html = "<table><thead><tr><th>Source</th><th>Pipeline Name</th><th>Threads</th><th>Instances</th></tr></thead><tbody>"
        for source, details in info.items():
            html += (
                f"<tr><td>{source}</td><td>{details.get('name','')}</td>"
                f"<td>{smart_format(details.get('threads',''))}</td>"
                f"<td>{smart_format(details.get('instances',''))}</td></tr>"
            )
        html += "</tbody></table>"
        return html_section("🔧 Pipeline Info", html)
    except Exception:
        return html_section("🔧 Pipeline Info", f"<p>⚠️ Could not parse JSON: {info_data}</p>")

def format_combined_topic_table(row):
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

def format_lag_table(row):
    html = (
        "<table><thead><tr><th>Min Lag</th><th>Avg Lag</th><th>Max Lag</th></tr></thead><tbody>"
        f"<tr><td>{smart_format(safe_get(row,'min_lag'))}</td>"
        f"<td>{smart_format(safe_get(row,'avg_lag'))}</td>"
        f"<td>{smart_format(safe_get(row,'max_lag'))}</td></tr>"
        "</tbody></table>"
    )
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

def format_kafka_specs(row):
    """Parse and format kafka_specs JSON into separate Input and Output topic tables (HTML)."""
    specs_data = safe_get(row, "kafka_specs")
    if not specs_data:
        return ""

    html = "<section><h2>🪣 Kafka Specs</h2>"
    try:
        specs = json.loads(specs_data)

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
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Test Report</title>
<style>
* {{
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}}

body {{
  font-family: -apple-system, BlinkMacSystemFont, 'Inter', 'Segoe UI', 'Roboto', 'Oxygen', 'Ubuntu', sans-serif;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  min-height: 100vh;
  padding: 2rem 1rem;
  color: #1e293b;
  line-height: 1.6;
}}

.container {{
  max-width: 1200px;
  margin: 0 auto;
}}

h1 {{
  color: #ffffff;
  font-size: 2.5rem;
  font-weight: 700;
  margin-bottom: 2rem;
  text-align: center;
  text-shadow: 0 2px 4px rgba(0,0,0,0.1);
  letter-spacing: -0.02em;
}}

h2 {{
  color: #0f172a;
  font-size: 1.5rem;
  font-weight: 600;
  margin-bottom: 1.25rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  letter-spacing: -0.01em;
}}

h3 {{
  color: #334155;
  font-size: 1.125rem;
  font-weight: 600;
  margin: 1.5rem 0 1rem 0;
  padding-bottom: 0.5rem;
  border-bottom: 2px solid #e2e8f0;
}}

section {{
  background: #ffffff;
  padding: 2rem;
  border-radius: 16px;
  box-shadow: 0 4px 6px -1px rgba(0,0,0,0.1), 0 2px 4px -1px rgba(0,0,0,0.06);
  margin-bottom: 1.5rem;
  border: 1px solid rgba(148, 163, 184, 0.1);
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}}

section:hover {{
  transform: translateY(-2px);
  box-shadow: 0 10px 15px -3px rgba(0,0,0,0.1), 0 4px 6px -2px rgba(0,0,0,0.05);
}}

table {{
  width: 100%;
  border-collapse: separate;
  border-spacing: 0;
  margin-top: 1rem;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 1px 3px 0 rgba(0,0,0,0.1);
}}

thead {{
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}}

th {{
  padding: 1rem;
  text-align: left;
  font-weight: 600;
  font-size: 0.875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #ffffff;
  border: none;
}}

td {{
  padding: 1rem;
  border-bottom: 1px solid #e2e8f0;
  font-size: 0.9375rem;
  color: #475569;
}}

tbody tr {{
  background: #ffffff;
  transition: background-color 0.15s ease;
}}

tbody tr:nth-child(even) {{
  background-color: #f8fafc;
}}

tbody tr:hover {{
  background-color: #f1f5f9;
  transform: scale(1.001);
}}

tbody tr:last-child td {{
  border-bottom: none;
}}

ul {{
  list-style-type: none;
  padding-left: 0;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1rem;
}}

li {{
  padding: 0.75rem 1rem;
  background: #f8fafc;
  border-radius: 8px;
  border-left: 3px solid #667eea;
  transition: all 0.2s ease;
}}

li:hover {{
  background: #f1f5f9;
  border-left-color: #764ba2;
  transform: translateX(4px);
}}

strong {{
  color: #0f172a;
  font-weight: 600;
  font-size: 0.875rem;
  text-transform: uppercase;
  letter-spacing: 0.025em;
}}

code {{
  background: linear-gradient(135deg, #667eea15 0%, #764ba215 100%);
  padding: 0.25rem 0.625rem;
  border-radius: 6px;
  font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Fira Code', 'Consolas', monospace;
  font-size: 0.9em;
  color: #5b21b6;
  border: 1px solid #ddd6fe;
  font-weight: 500;
}}

@media (max-width: 768px) {{
  body {{
    padding: 1rem 0.5rem;
  }}
  
  h1 {{
    font-size: 1.75rem;
  }}
  
  section {{
    padding: 1.25rem;
    border-radius: 12px;
  }}
  
  ul {{
    grid-template-columns: 1fr;
  }}
  
  table {{
    font-size: 0.875rem;
  }}
  
  th, td {{
    padding: 0.75rem 0.5rem;
  }}
}}

@media print {{
  body {{
    background: white;
    padding: 0;
  }}
  
  section {{
    box-shadow: none;
    border: 1px solid #e2e8f0;
    page-break-inside: avoid;
  }}
  
  section:hover {{
    transform: none;
  }}
}}
</style>
</head>
<body>
<div class="container">
{content}
</div>
</body>
</html>
"""

def generate_html_report(row):
    html_content = (
        f"<h1>📊 Test Report - {safe_get(row, 'test_id')}</h1>"
        + format_summary(row)
        + format_kafka_specs(row)
        + format_combined_topic_table(row)
        + format_lag_table(row)
    )

    html_content += "<h2>🖥️ Pod Metrics</h2>"
    for title, group in POD_GROUPS.items():
        html_content += format_grouped_table(title, group, row)

    html_content += "<h2>🔧 Pipeline Pod Metrics</h2>" + format_pipeline_info(row)
    for title, group in PIPELINE_GROUPS.items():
        html_content += format_grouped_table(title, group, row)

    html_content += "<h2>💻 Node Metrics</h2>"
    for title, group in NODE_GROUPS.items():
        html_content += format_grouped_table(title, group, row)

    return wrap_html_template(html_content)

def main():
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    conn = sqlite3.connect(DB_PATH)
    df = pd.read_sql_query(f"SELECT * FROM {TABLE_NAME}", conn)
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

    print("\n✅ All HTML reports generated successfully in 'html_reports' folder.")

if __name__ == "__main__":
    main()
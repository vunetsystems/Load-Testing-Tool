import os
import sqlite3
import pandas as pd
import json

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
# HELPER FUNCTIONS
# ===============================

def safe_get(row, key):
    return row[key] if key in row and pd.notna(row[key]) else ""

def html_section(title, content):
    """Wrap content in a styled HTML section."""
    return f"<section><h2>{title}</h2>{content}</section>"

def format_summary(row):
    html = "<ul>"
    for field in SUMMARY_FIELDS:
        value = safe_get(row, field)
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
        html = (
            "<table><thead><tr>"
            "<th>Source</th><th>Pipeline Name</th><th>Threads</th><th>Instances</th>"
            "</tr></thead><tbody>"
        )
        for source, details in info.items():
            html += f"<tr><td>{source}</td><td>{details.get('name','')}</td>"
            html += f"<td>{details.get('threads','')}</td><td>{details.get('instances','')}</td></tr>"
        html += "</tbody></table>"
        return html_section("🔧 Pipeline Info", html)
    except Exception:
        return html_section("🔧 Pipeline Info", f"<p>⚠️ Could not parse JSON: {info_data}</p>")

def format_combined_topic_table(row):
    html = (
        "<table><thead><tr><th>Type</th><th>Min (msg/s)</th><th>Avg (msg/s)</th><th>Max (msg/s)</th></tr></thead><tbody>"
        f"<tr><td>Input</td><td>{safe_get(row,'min_input_msgs_per_sec')}</td><td>{safe_get(row,'avg_input_msgs_per_sec')}</td><td>{safe_get(row,'max_input_msgs_per_sec')}</td></tr>"
        f"<tr><td>Output</td><td>{safe_get(row,'min_output_msgs_per_sec')}</td><td>{safe_get(row,'avg_output_msgs_per_sec')}</td><td>{safe_get(row,'max_output_msgs_per_sec')}</td></tr>"
        "</tbody></table>"
    )
    return html_section("📈 Input/Output Topic Metrics", html)

def format_lag_table(row):
    html = (
        "<table><thead><tr><th>Min Lag</th><th>Avg Lag</th><th>Max Lag</th></tr></thead><tbody>"
        f"<tr><td>{safe_get(row,'min_lag')}</td><td>{safe_get(row,'avg_lag')}</td><td>{safe_get(row,'max_lag')}</td></tr>"
        "</tbody></table>"
    )
    return html_section("⏱️ Lag Metrics", html)

def extract_pipeline_name_from_info(row):
    """Safely extract the first pipeline name from pipeline_info JSON in the row."""
    info_data = safe_get(row, "pipeline_info")
    if not info_data:
        return None
    try:
        info = json.loads(info_data)
        if isinstance(info, dict) and len(info) > 0:
            first_entry = next(iter(info.values()))
            name = first_entry.get("name")
            if name:
                return name
    except Exception:
        pass
    return None

def format_grouped_table(title, group_def, row):
    """
    Create grouped HTML table (Pods / Nodes / Pipelines).
    For pipeline groups, show the pipeline name (from pipeline_info) instead of 'pipeline_pod'.
    """
    is_cpu = "CPU" in title
    is_memory = "Memory" in title
    unit = "Cores" if is_cpu else "Gi"
    allocated_label = f"Allocated ({unit})"

    # determine if pipeline group
    is_pipeline_group = "Pipeline Pod" in title

    # prepare pipeline name if needed
    pipeline_name = None
    if is_pipeline_group:
        pipeline_name = extract_pipeline_name_from_info(row)
        if not pipeline_name:
            pipeline_name = "unknown_pipeline"

    # Build table
    html = (
        f"<table><thead><tr><th>Component</th><th>{allocated_label}</th><th>Min (%)</th><th>Avg (%)</th><th>Max (%)</th></tr></thead><tbody>"
    )

    for comp in group_def["rows"]:
        # allocated field pattern
        alloc_field = f"{comp}_{'cpu' if is_cpu else 'mem'}_allocated"
        min_field, avg_field, max_field = [f"{comp}_{col}" for col in group_def["cols"]]

        # label shown in first column
        if is_pipeline_group:
            display_comp = pipeline_name
        else:
            display_comp = comp

        html += (
            f"<tr><td>{display_comp}</td>"
            f"<td>{safe_get(row, alloc_field)}</td>"
            f"<td>{safe_get(row, min_field)}</td>"
            f"<td>{safe_get(row, avg_field)}</td>"
            f"<td>{safe_get(row, max_field)}</td></tr>"
        )

    html += "</tbody></table>"
    return html_section(title, html)

def format_kafka_specs(row):
    specs_data = safe_get(row, "kafka_specs")
    if not specs_data:
        return ""
    try:
        specs = json.loads(specs_data)
        html = ""
        if "input_topics" in specs:
            html += "<h4>📥 Input Topics</h4><table><thead><tr><th>Name</th><th>Partitions</th><th>Replication</th></tr></thead><tbody>"
            for t in specs["input_topics"]:
                html += f"<tr><td>{t.get('name','')}</td><td>{t.get('partitions','')}</td><td>{t.get('replication_factor','')}</td></tr>"
            html += "</tbody></table>"
        if "output_topics" in specs:
            html += "<h4>📤 Output Topics</h4><table><thead><tr><th>Name</th><th>Partitions</th><th>Replication</th></tr></thead><tbody>"
            for t in specs["output_topics"]:
                html += f"<tr><td>{t.get('name','')}</td><td>{t.get('partitions','')}</td><td>{t.get('replication_factor','')}</td></tr>"
            html += "</tbody></table>"
        return html_section("🪄 Kafka Specs", html)
    except Exception:
        return html_section("🪄 Kafka Specs", f"<p>⚠️ Could not parse JSON: {specs_data}</p>")

def get_o11y_source_name(o11y_sources):
    try:
        src = json.loads(o11y_sources)
        if isinstance(src, list) and len(src) > 0:
            return src[0].replace(" ", "_")
    except Exception:
        pass
    return "unknown_source"

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

def wrap_html_template(content):
    """Wrap report body with full HTML structure and CSS."""
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
  font-size: 0.95em;
}}
</style>
</head>
<body>
{content}
</body>
</html>
"""

def main():
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    conn = sqlite3.connect(DB_PATH)
    df = pd.read_sql_query(f"SELECT * FROM {TABLE_NAME}", conn)
    conn.close()
    print(f"✅ Loaded {len(df)} rows from database.\n")

    for _, row in df.iterrows():
        o11y_source = get_o11y_source_name(str(row["o11y_sources"]))
        eps = safe_get(row, "target_eps")
        file_name = f"{o11y_source}_{eps}.html"
        file_path = os.path.join(OUTPUT_DIR, file_name)
        html_content = generate_html_report(row)

        with open(file_path, "w", encoding="utf-8") as f:
            f.write(html_content)
        print(f"📝 Generated: {file_path}")

    print("\n✅ All HTML reports generated successfully in 'html_reports' folder.")

if __name__ == "__main__":
    main()

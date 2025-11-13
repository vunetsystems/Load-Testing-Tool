import os
import sqlite3
import pandas as pd
import json

# ===============================
# CONFIGURATION
# ===============================
DB_PATH = "vudatasim.db"      # <-- Replace with your SQLite DB path
TABLE_NAME = "test_runs"       # <-- Replace with your actual table name
OUTPUT_DIR = "reports"


# ===============================
# FIELD GROUP DEFINITIONS
# ===============================

SUMMARY_FIELDS = [
    "test_id", "target_eps", "start_time", "end_time",
    "o11y_sources", "timeout_seconds", "status"
]

# Pods first
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


# Nodes next
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
    """Return value if key exists, else an empty string."""
    return row[key] if key in row and pd.notna(row[key]) else ""


def format_summary(row):
    md = "## 🧾 Test Summary\n\n"
    for field in SUMMARY_FIELDS:
        value = safe_get(row, field)

        # Special handling for timeout_seconds → Duration (hours)
        if field == "timeout_seconds":
            if value:
                try:
                    hours = float(value) / 3600
                    value = f"{hours:.2f} hours"
                except Exception:
                    pass
            label = "Duration"  # Rename field
        else:
            label = field.replace("_", " ").title()

        md += f"- **{label}**: `{value}`\n"

    md += "\n"
    return md

def format_pipeline_info(row):
    """Format pipeline_info JSON into a readable Markdown table."""
    info_data = safe_get(row, "pipeline_info")
    if not info_data:
        return ""

    md = "### Pipeline Info\n\n"

    try:
        info = json.loads(info_data)
        if not isinstance(info, dict):
            md += f"⚠️ Unexpected data format: `{info_data}`\n\n"
            return md

        # Build table header
        md += "| Source | Pipeline Name | Threads | Instances |\n"
        md += "|---------|----------------|----------|------------|\n"

        # Each key is a pipeline source (e.g., "LinuxMonitor", "Kubernetes", "Mssql")
        for source, details in info.items():
            name = details.get("name", "")
            threads = details.get("threads", "")
            instances = details.get("instances", "")
            md += f"| `{source}` | `{name}` | `{threads}` | `{instances}` |\n"

        md += "\n"

    except Exception as e:
        md += f"⚠️ Could not parse pipeline_info JSON: `{info_data}`\n\n"

    return md





def format_combined_topic_table(row):
    """Combine input/output message rates into one horizontal table."""
    md = "### Input/Output Topic Metrics\n\n"
    md += "| Type | Min (msg/s) | Avg (msg/s) | Max (msg/s) |\n"
    md += "|------|--------------|--------------|--------------|\n"

    md += f"| Input  | `{safe_get(row, 'min_input_msgs_per_sec')}` | `{safe_get(row, 'avg_input_msgs_per_sec')}` | `{safe_get(row, 'max_input_msgs_per_sec')}` |\n"
    md += f"| Output | `{safe_get(row, 'min_output_msgs_per_sec')}` | `{safe_get(row, 'avg_output_msgs_per_sec')}` | `{safe_get(row, 'max_output_msgs_per_sec')}` |\n\n"
    return md


def format_lag_table(row):
    """Lag metrics table (inverted horizontally)."""
    md = "### Lag Metrics\n\n"
    md += "| Min Lag | Avg Lag | Max Lag |\n"
    md += "|----------|----------|----------|\n"
    md += f"| `{safe_get(row, 'min_lag')}` | `{safe_get(row, 'avg_lag')}` | `{safe_get(row, 'max_lag')}` |\n\n"
    return md



def format_grouped_table(title, group_def, row):
    """
    Create grouped horizontal table (Pods / Nodes / Pipelines) 
    with Allocated column (raw numbers) and percentage metrics for Min/Avg/Max.
    """
    is_pipeline_group = "Pipeline Pod" in title
    is_pod_or_node_group = any(keyword in title for keyword in ["Pod", "Node"])

    # Determine if this group is CPU or Memory
    is_cpu = "CPU" in title
    is_memory = "Memory" in title

    # Add % marker to resource-based titles
    if is_pod_or_node_group and (is_cpu or is_memory):
        title_suffix = " (%)"
    else:
        title_suffix = ""

    # Extract pipeline name (for pipeline groups)
    pipeline_name = None
    if is_pipeline_group:
        try:
            info = json.loads(safe_get(row, "pipeline_info"))
            if isinstance(info, dict) and len(info) > 0:
                first_entry = next(iter(info.values()))
                pipeline_name = first_entry.get("name", "unknown_pipeline")
        except Exception:
            pipeline_name = "unknown_pipeline"

    # Table column label (Pipeline or Component)
    col_label = "Pipeline Name" if is_pipeline_group else "Component"

    # Header construction with optional % markers
    header_allocated = "Allocated"  # Raw number (not %)
    header_min = "Min (%)" if is_pod_or_node_group else "Min"
    header_avg = "Avg (%)" if is_pod_or_node_group else "Avg"
    header_max = "Max (%)" if is_pod_or_node_group else "Max"

    # Build Markdown table
    md = f"### {title}{title_suffix}\n\n"
    md += f"| {col_label} | {header_allocated} | {header_min} | {header_avg} | {header_max} |\n"
    md += "|--------------|-------------|----------|----------|----------|\n"

    # Function to determine allocated field key
    def get_allocated_field(component):
        if is_cpu:
            return f"{component}_cpu_allocated"
        elif is_memory:
            return f"{component}_mem_allocated"
        return None

    # For pipeline pods, only one row (pipeline name)
    if is_pipeline_group:
        for comp in group_def["rows"]:
            alloc_field = get_allocated_field(comp)
            min_field = f"{comp}_{group_def['cols'][0]}"
            avg_field = f"{comp}_{group_def['cols'][1]}"
            max_field = f"{comp}_{group_def['cols'][2]}"
            md += (
                f"| `{pipeline_name}` | `{safe_get(row, alloc_field)}` | "
                f"`{safe_get(row, min_field)}` | `{safe_get(row, avg_field)}` | `{safe_get(row, max_field)}` |\n"
            )
    else:
        # Normal pod/node tables
        for comp in group_def["rows"]:
            alloc_field = get_allocated_field(comp)
            min_field = f"{comp}_{group_def['cols'][0]}"
            avg_field = f"{comp}_{group_def['cols'][1]}"
            max_field = f"{comp}_{group_def['cols'][2]}"
            md += (
                f"| {comp} | `{safe_get(row, alloc_field)}` | "
                f"`{safe_get(row, min_field)}` | `{safe_get(row, avg_field)}` | `{safe_get(row, max_field)}` |\n"
            )

    md += "\n"
    return md





def get_o11y_source_name(o11y_sources):
    """Extract clean o11y source name from JSON string like '["Linux Monitor"]'."""
    try:
        src = json.loads(o11y_sources)
        if isinstance(src, list) and len(src) > 0:
            return src[0].replace(" ", "_")
    except Exception:
        pass
    return "unknown_source"

def format_kafka_specs(row):
    """Parse and format kafka_specs JSON into separate Input and Output topic tables."""
    specs_data = safe_get(row, "kafka_specs")
    if not specs_data:
        return ""

    md = "### Kafka Specs\n\n"
    try:
        specs = json.loads(specs_data)

        # --- Input Topics ---
        if "input_topics" in specs and isinstance(specs["input_topics"], list):
            md += "#### 📥 Input Topics\n\n"
            md += "| Topic Name | Partitions | Replication Factor |\n"
            md += "|-------------|-------------|--------------------|\n"
            for topic in specs["input_topics"]:
                name = topic.get("name", "")
                partitions = topic.get("partitions", "")
                replication = topic.get("replication_factor", "")
                md += f"| `{name}` | `{partitions}` | `{replication}` |\n"
            md += "\n"

        # --- Output Topics ---
        if "output_topics" in specs and isinstance(specs["output_topics"], list):
            md += "#### 📤 Output Topics\n\n"
            md += "| Topic Name | Partitions | Replication Factor |\n"
            md += "|-------------|-------------|--------------------|\n"
            for topic in specs["output_topics"]:
                name = topic.get("name", "")
                partitions = topic.get("partitions", "")
                replication = topic.get("replication_factor", "")
                md += f"| `{name}` | `{partitions}` | `{replication}` |\n"
            md += "\n"

    except Exception as e:
        md += f"⚠️ Could not parse kafka_specs JSON: `{specs_data}`\n"

    return md + "\n"




def generate_report(row):
    """Generate the markdown for one test record."""
    md = f"# 📊 Test Report - {safe_get(row, 'test_id')}\n\n"
    md += format_summary(row)

    # Combined Topic Metrics section
    md += "## 📈 Topic Metrics\n\n"
    md += format_kafka_specs(row)  
    md += format_combined_topic_table(row)
    md += format_lag_table(row)

    # Pod Metrics FIRST
    md += "## 🖥️ Pod Metrics\n\n"
    for title, group_def in POD_GROUPS.items():
        md += format_grouped_table(title, group_def, row)

    # Pipeline Pod Metrics (separate section)
    md += "## 🔧 Pipeline Pod Metrics\n\n"
    md += format_pipeline_info(row)

    for title, group_def in PIPELINE_GROUPS.items():
        md += format_grouped_table(title, group_def, row)

    # Node Metrics SECOND
    md += "## 💻 Node Metrics\n\n"
    for title, group_def in NODE_GROUPS.items():
        md += format_grouped_table(title, group_def, row)

    return md


# ===============================
# MAIN SCRIPT
# ===============================

def main():
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    conn = sqlite3.connect(DB_PATH)
    df = pd.read_sql_query(f"SELECT * FROM {TABLE_NAME}", conn)
    conn.close()

    print(f"✅ Loaded {len(df)} rows from database.\n")

    for _, row in df.iterrows():
        o11y_source = get_o11y_source_name(str(row["o11y_sources"]))
        eps = safe_get(row, "target_eps")
        file_name = f"{o11y_source}_{eps}.md"
        file_path = os.path.join(OUTPUT_DIR, file_name)

        md_content = generate_report(row)
        with open(file_path, "w", encoding="utf-8") as f:
            f.write(md_content)

        print(f"📝 Generated: {file_path}")

    print("\n✅ All Markdown reports generated successfully in the 'reports' folder.")


if __name__ == "__main__":
    main()

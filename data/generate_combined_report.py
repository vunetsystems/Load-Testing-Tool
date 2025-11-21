import os
import sqlite3
import pandas as pd
import json
import re
from pathlib import Path

# ===============================
# CONFIGURATION
# ===============================
VUDATASIM_DB = "vudatasim.db"
VUDATASIM_TABLE = "test_runs"
K6_TABLE = "k6_runs"
OUTPUT_DIR = "combined_reports"

# Import the existing report generators
# Script names: generate_html_reports.py and k6.py
try:
    import generate_html_reports
    import k6
except ImportError:
    print("⚠️ Make sure generate_html_reports.py and k6.py are in the same directory")
    exit(1)

# ===============================
# HELPER FUNCTIONS
# ===============================

def normalize_test_name(name):
    """
    Normalize test name by removing suffixes like 'vudatasim', 'k6', etc.
    Example: 'AWS_ALB 5K EPS vudatasim' -> 'AWS_ALB 5K EPS'
    """
    if not name:
        return ""
    
    # Remove common suffixes
    suffixes = ['vudatasim', 'k6', 'test']
    name_lower = name.lower().strip()
    
    for suffix in suffixes:
        if name_lower.endswith(suffix):
            name = name[:-(len(suffix))].strip()
            break
    
    return name.strip()

def get_test_pairs():
    """
    Match vudatasim and k6 tests based on normalized names.
    Returns list of tuples: (normalized_name, vudatasim_row, k6_row)
    """
    conn = sqlite3.connect(VUDATASIM_DB)
    
    # Load vudatasim tests
    vuda_df = pd.read_sql_query(f"SELECT * FROM {VUDATASIM_TABLE} WHERE report_generated = 0", conn)

    # Load k6 tests with required fields
    k6_query = f"""
        SELECT
            test_id, test_name, status, duration, o11y_sources,
            start_time, end_time, vus,
            summarised,
            Metrics_Login,
            Overall_Dashboard_Load_Times,
            Panel_Performance_Breakdown
        FROM {K6_TABLE}
        WHERE
            summarised IS NOT NULL AND summarised != '' AND
            Metrics_Login IS NOT NULL AND Metrics_Login != '' AND
            report_generated = 0
    """
    k6_df = pd.read_sql_query(k6_query, conn)
    conn.close()
    
    # Create lookup dictionaries
    vuda_dict = {}
    for _, row in vuda_df.iterrows():
        test_name = row.get('test_name', '')
        normalized = normalize_test_name(test_name)
        if normalized:
            vuda_dict[normalized] = row
    
    k6_dict = {}
    for _, row in k6_df.iterrows():
        test_name = row.get('test_name', '')
        normalized = normalize_test_name(test_name)
        if normalized:
            k6_dict[normalized] = row
    
    # Find matches
    matched_pairs = []
    all_names = set(vuda_dict.keys()) | set(k6_dict.keys())
    
    for name in all_names:
        vuda_row = vuda_dict.get(name)
        k6_row = k6_dict.get(name)
        
        # Only include if at least one exists
        if vuda_row is not None or k6_row is not None:
            matched_pairs.append((name, vuda_row, k6_row))
    
    return matched_pairs

def generate_combined_html(normalized_name, vuda_row, k6_row):
    """
    Generate combined HTML with toggle between vudatasim and k6 reports.
    """
    # Generate individual report contents
    vuda_content = ""
    k6_content = ""
    
    if vuda_row is not None:
        # Generate vudatasim report content (without wrapper)
        vuda_html = generate_html_reports.generate_html_report(vuda_row)
        # Extract body content (remove html wrapper)
        vuda_content = extract_body_content(vuda_html)
    
    if k6_row is not None:
        # Generate k6 report content (without wrapper)
        try:
            summary_raw = k6.get_json_str(k6_row.get("summarised"), default_fallback="{}")
            login_raw = k6.get_json_str(k6_row.get("Metrics_Login"), default_fallback="{}")
            dashboard_raw = k6.get_json_str(k6_row.get("Overall_Dashboard_Load_Times"), default_fallback="[]")
            panel_raw = k6.get_json_str(k6_row.get("Panel_Performance_Breakdown"), default_fallback="[]")
            
            login_data = json.loads(login_raw)
            dashboard_data = json.loads(dashboard_raw)
            panel_data = json.loads(panel_raw)

            html_summary = k6.format_summary(k6_row)
            html_login = k6.format_login(login_data)
            html_dashboard = k6.format_dashboard(dashboard_data)
            html_panel = k6.format_panels(panel_data)
            
            k6_content = html_summary + html_login + html_dashboard + html_panel
        except Exception as e:
            k6_content = f"<p>⚠️ Error generating K6 report: {e}</p>"
    
    # Build combined HTML with toggle
    return build_combined_template(normalized_name, vuda_content, k6_content, 
                                   has_vuda=(vuda_row is not None), 
                                   has_k6=(k6_row is not None))

def extract_body_content(html):
    """Extract content between <body> tags."""
    try:
        start = html.find('<body>') + 6
        end = html.find('</body>')
        if start > 5 and end > start:
            return html[start:end]
    except:
        pass
    return html

def build_combined_template(name, vuda_content, k6_content, has_vuda, has_k6):
    """
    Build HTML template with toggle functionality.
    """
    # Determine which report to show by default
    default_view = 'vudatasim' if has_vuda else 'k6'
    
    toggle_buttons = ""
    if has_vuda and has_k6:
        toggle_buttons = """
        <div class="toggle-container">
            <button class="toggle-btn active" onclick="showReport('vudatasim')">
                🖥️ Infra
            </button>
            <button class="toggle-btn" onclick="showReport('k6')">
                🎨 UI
            </button>
        </div>
        """
    elif has_vuda:
        toggle_buttons = "<div class='toggle-container'><p style='text-align:center; font-weight:600;'>🖥️ Infra</p></div>"
    elif has_k6:
        toggle_buttons = "<div class='toggle-container'><p style='text-align:center; font-weight:600;'>🎨 UI</p></div>"
    
    vuda_display = "block" if default_view == 'vudatasim' else "none"
    k6_display = "block" if default_view == 'k6' else "none"
    
    return f"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Test Report: {name}</title>
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

.toggle-container {{
    display: flex;
    gap: 1rem;
    margin-bottom: 1.5rem;
    background: white;
    padding: 1rem;
    border-radius: 10px;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
    justify-content: center;
}}

.toggle-btn {{
    padding: 0.75rem 2rem;
    font-size: 1rem;
    font-weight: 600;
    border: 2px solid #e2e8f0;
    background: white;
    color: #4a5568;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s ease;
}}

.toggle-btn:hover {{
    background: #f7fafc;
    border-color: #1a365d;
}}

.toggle-btn.active {{
    background: #1a365d;
    color: white;
    border-color: #1a365d;
}}

.report-container {{
    display: none;
}}

.report-container.active {{
    display: block;
    animation: fadeIn 0.3s ease-in;
}}

@keyframes fadeIn {{
    from {{ opacity: 0; }}
    to {{ opacity: 1; }}
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

.desc {{
    background: #f0f8ff;
    border-left: 3px solid #1a365d;
    padding: 8px 12px;
    margin: .5rem 0;
    font-size: .9em;
}}

.chart-container {{
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
    margin-top: 1rem;
}}

.chart {{
    flex: 1;
    min-width: 400px;
    box-shadow: 0 1px 3px rgba(0,0,0,.1);
    border-radius: 8px;
    background: #fff;
    padding: .5rem;
}}

.chart-full {{
    width: 100%;
    box-shadow: 0 1px 3px rgba(0,0,0,.1);
    border-radius: 8px;
    background: #fff;
    margin-top: 1rem;
    padding: .5rem;
}}

.no-report {{
    text-align: center;
    padding: 3rem;
    color: #718096;
    font-style: italic;
}}
</style>
</head>
<body>

<h1>📊 Performance Report - {name}</h1>

{toggle_buttons}

<div id="vudatasim-report" class="report-container {'active' if default_view == 'vudatasim' else ''}">
    {vuda_content if has_vuda else '<div class="no-report">VUDataSim report not available</div>'}
</div>

<div id="k6-report" class="report-container {'active' if default_view == 'k6' else ''}">
    {k6_content if has_k6 else '<div class="no-report">K6 report not available</div>'}
</div>

<script>
function showReport(reportType) {{
    // Hide all reports
    document.querySelectorAll('.report-container').forEach(el => {{
        el.classList.remove('active');
    }});
    
    // Remove active class from all buttons
    document.querySelectorAll('.toggle-btn').forEach(btn => {{
        btn.classList.remove('active');
    }});
    
    // Show selected report
    const reportId = reportType + '-report';
    document.getElementById(reportId).classList.add('active');
    
    // Activate clicked button
    event.target.classList.add('active');
}}
</script>

</body>
</html>"""

# ===============================
# MAIN EXECUTION
# ===============================

def main():
    print("🚀 Starting Combined Report Generator\n")
    
    # Create output directory
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    
    # Get matched test pairs
    test_pairs = get_test_pairs()

    if not test_pairs:
        print("⚠️ No unprocessed test pairs found. Make sure both databases have matching test names with report_generated = 0.")
        return

    print(f"✅ Found {len(test_pairs)} unprocessed test pair(s)\n")

    # Open connection for updates
    conn = sqlite3.connect(VUDATASIM_DB)

    # Generate combined reports
    for normalized_name, vuda_row, k6_row in test_pairs:
        print(f"📝 Processing: {normalized_name}")
        
        try:
            # Generate combined HTML
            combined_html = generate_combined_html(normalized_name, vuda_row, k6_row)
            
            # Create safe filename
            safe_name = "".join(c if c.isalnum() or c in (' ', '_', '-') else '_' 
                              for c in normalized_name).strip()
            file_name = f"{safe_name}.html"
            file_path = os.path.join(OUTPUT_DIR, file_name)
            
            # Write to file
            with open(file_path, "w", encoding="utf-8") as f:
                f.write(combined_html)

            print(f"   ✓ Generated: {file_path}")

            # Update report_generated to 1
            if vuda_row is not None:
                conn.execute("UPDATE test_runs SET report_generated = 1 WHERE test_id = ?", (vuda_row['test_id'],))
            if k6_row is not None:
                conn.execute("UPDATE k6_runs SET report_generated = 1 WHERE test_id = ?", (k6_row['test_id'],))
            conn.commit()

            # Show which reports were included
            reports = []
            if vuda_row is not None:
                reports.append("VUDataSim")
            if k6_row is not None:
                reports.append("K6")
            print(f"   ℹ️ Includes: {', '.join(reports)}\n")
            
        except Exception as e:
            print(f"   ❌ Error generating report: {e}\n")

    conn.close()
    print(f"\n✅ All combined reports generated in '{OUTPUT_DIR}' folder")
    print(f"📊 Total reports: {len(test_pairs)}")

if __name__ == "__main__":
    main()
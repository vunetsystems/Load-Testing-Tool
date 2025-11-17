import os
import sqlite3
import pandas as pd
import json
import math
import plotly.express as px
import plotly.io as pio

# ===============================
# CONFIGURATION
# ===============================
DB_PATH = "vudatasim.db"
TABLE_NAME = "k6_runs" 
OUTPUT_DIR = "k6_html_reports"

# ===============================
# HELPER FUNCTIONS
# ===============================

def smart_format(value, precision=2):
    """Round only if float and not an integer. Keep integers unchanged."""
    try:
        num = float(value)
        if num.is_integer():
            return str(int(num))
        return f"{num:.{precision}f}"
    except (ValueError, TypeError, AttributeError):
        return str(value)

def get_json_str(value, default_fallback):
    """
    Safely casts a value to a string and checks if it's a simple non-JSON 
    value before returning it for json.loads. This fixes the 'not int' error.
    """
    try:
        s = str(value).strip()
    except:
        return default_fallback
        
    # Check for simple values often stored instead of a full JSON string
    if s.lower() in ('none', 'nan', 'true', 'false', '0', '1', ''):
        return default_fallback
    return s

def wrap_html_template(content, test_id):
    """Wraps the generated content in a full HTML5 page with improved CSS."""
    return f"""
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>K6 Report: {test_id}</title>
<style>
body {{
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
  margin: 2rem auto;
  max-width: 1200px;
  background: #f4f7f9; /* Light grey background */
  color: #1a202c;
  line-height: 1.6;
}}
h1 {{ 
    color: #004d40; /* Dark Teal */
    border-bottom: 3px solid #b2dfdb; 
    padding-bottom: 12px; 
    font-weight: 800;
    margin-bottom: 2rem;
}}
h2 {{ 
    color: #1a365d; /* Dark Blue */
    border-bottom: 1px solid #e0e6ed; 
    padding-bottom: 5px; 
    margin-top: 2.5rem; 
    font-size: 1.6rem;
}}
h4 {{
    font-weight: 600;
    margin-top: 1.5rem;
}}
section {{
  background: #ffffff;
  padding: 1.5rem 2rem;
  border-radius: 12px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.08);
  margin-bottom: 2rem;
}}
table {{
  width: 100%;
  border-collapse: separate; /* Use separate for rounded corners */
  border-spacing: 0;
  margin-top: 1.5rem;
  overflow: hidden; /* Helps with rounded corners on headers */
  border-radius: 8px;
}}
th, td {{
  border: none;
  padding: 12px 15px;
  text-align: left;
}}
th {{
  background: #00796b; /* Mid Teal Header */
  color: white;
  font-weight: 700;
}}
tr:nth-child(even) {{
  background-color: #f8fcfd; /* Very light blue stripe */
}}
tr:hover {{
  background-color: #e3f2fd; /* Lightest blue on hover */
}}
td:first-child {{
    font-weight: 500;
    color: #2d3748;
}}
code {{
  background-color: #e6fffa; /* Pale green/teal for code */
  color: #004d40;
  padding: 3px 8px;
  border-radius: 6px;
  font-size: 0.9em;
}}
.chart-container {{
    display: flex;
    flex-wrap: wrap;
    justify-content: space-between;
    gap: 20px;
    margin-top: 1.5rem;
}}
.chart {{
    width: 49%;
    min-width: 450px;
    box-shadow: 0 2px 5px rgba(0,0,0,0.05);
    border-radius: 8px;
    background: #ffffff;
}}
.chart-full {{
    width: 100%;
    min-width: 600px;
    box-shadow: 0 2px 5px rgba(0,0,0,0.05);
    border-radius: 8px;
    background: #ffffff;
    margin-top: 1.5rem;
}}
</style>
</head>
<body>
<h1>📊 K6 Scale & Performance Report: <code>{test_id}</code></h1>
{content}
</body>
</html>
"""

def format_summary(row):
    """
    Generates HTML for the Test Configuration section.
    Updated to include VUS and use DB columns for start/end time.
    """
    # Use database columns directly
    start_time_val = row.get('start_time', 'N/A')
    end_time_val = row.get('end_time', 'N/A')
    vus_val = row.get('vus', 'N/A')
    
    html = "<section><h2>Test Configuration</h2><table>"
    
    html += f"<tr><td><strong>Test ID</strong></td><td><code>{row.get('test_id', 'N/A')}</code></td></tr>"
    html += f"<tr><td><strong>Test Name</strong></td><td>{row.get('test_name', 'N/A')}</td></tr>"
    html += f"<tr><td><strong>Status</strong></td><td>{row.get('status', 'N/A')}</td></tr>"
    html += f"<tr><td><strong>Virtual Users (VUS)</strong></td><td>{smart_format(vus_val, 0)}</td></tr>"
    html += f"<tr><td><strong>Start Time</strong></td><td>{start_time_val}</td></tr>"
    html += f"<tr><td><strong>End Time</strong></td><td>{end_time_val}</td></tr>"
    html += f"<tr><td><strong>Configured Duration</strong></td><td>{row.get('duration', 'N/A')}</td></tr>"
    html += f"<tr><td><strong>O11y Sources</strong></td><td>{row.get('o11y_sources', 'N/A')}</td></tr>"
    
    # Segment details are now omitted as requested.

    html += "</table></section>"
    return html

def create_login_charts(login_data):
    """Generates Plotly bar charts for login metrics."""
    try:
        df = pd.DataFrame({
            'Segment': ['Segment 1', 'Segment 2'],
            'Avg. Response Time (ms)': [
                float(login_data.get('seg1_avg_rt', 0)), 
                float(login_data.get('seg2_avg_rt', 0))
            ],
            'Success Rate (%)': [
                float(login_data.get('seg1_success_rate', 0)), 
                float(login_data.get('seg2_success_rate', 0))
            ]
        })

        fig_rt = px.bar(df, 
                        x='Segment', 
                        y='Avg. Response Time (ms)', 
                        title='Login Avg. Response Time',
                        text='Avg. Response Time (ms)',
                        color_discrete_sequence=['#1a365d'])
        fig_rt.update_traces(texttemplate='%{text:.2f} ms', textposition='outside')
        fig_rt.update_layout(uniformtext_minsize=8)

        fig_sr = px.bar(df, 
                        x='Segment', 
                        y='Success Rate (%)', 
                        title='Login Success Rate', 
                        range_y=[0, 100],
                        text='Success Rate (%)',
                        color_discrete_sequence=['#38a169'])
        fig_sr.update_traces(texttemplate='%{text:.2f}%', textposition='outside')
        fig_sr.update_layout(uniformtext_minsize=8)

        # Include Plotly.js on the first chart only
        chart_html = "<div class='chart-container'>"
        chart_html += f"<div class='chart'>{pio.to_html(fig_rt, full_html=False, include_plotlyjs='cdn')}</div>"
        chart_html += f"<div class='chart'>{pio.to_html(fig_sr, full_html=False)}</div>"
        chart_html += "</div>"
        return chart_html
    except Exception as e:
        return f"<p>⚠️ Could not generate login charts: {e}</p>"

def format_login(login_data):
    """Generates HTML for the Login Performance section."""
    html = "<section><h2>1. 🔑 Login Performance</h2>"
    
    html += create_login_charts(login_data)
    
    html += "<table>"
    html += "<thead><tr><th>Metric</th><th>Segment 1 (Sequential)</th><th>Segment 2 (Parallel)</th><th>Overall</th></tr></thead>"
    html += "<tbody>"
    html += f"<tr><td><strong>Total Attempts</strong></td><td>{smart_format(login_data.get('seg1_attempts', 'N/A'))}</td><td>{smart_format(login_data.get('seg2_attempts', 'N/A'))}</td><td>{smart_format(login_data.get('overall_attempts', 'N/A'))}</td></tr>"
    html += f"<tr><td><strong>Success Rate (%)</strong></td><td>{smart_format(login_data.get('seg1_success_rate', 'N/A'))}%</td><td>{smart_format(login_data.get('seg2_success_rate', 'N/A'))}%</td><td>{smart_format(login_data.get('overall_success_rate', 'N/A'))}%</td></tr>"
    html += f"<tr><td><strong>Avg. Response Time (ms)</strong></td><td>{smart_format(login_data.get('seg1_avg_rt', 'N/A'))}</td><td>{smart_format(login_data.get('seg2_avg_rt', 'N/A'))}</td><td>{smart_format(login_data.get('overall_avg_rt', 'N/A'))}</td></tr>"
    html += f"<tr><td><strong>P95 Response Time (ms)</strong></td><td>{smart_format(login_data.get('seg1_p95_rt', 'N/A'))}</td><td>{smart_format(login_data.get('seg2_p95_rt', 'N/A'))}</td><td>{smart_format(login_data.get('overall_p95_rt', 'N/A'))}</td></tr>"
    html += f"<tr><td><strong>P99 Response Time (ms)</strong></td><td>{smart_format(login_data.get('seg1_p99_rt', 'N/A'))}</td><td>{smart_format(login_data.get('seg2_p99_rt', 'N/A'))}</td><td>{smart_format(login_data.get('overall_p99_rt', 'N/A'))}</td></tr>"
    html += f"<tr><td><strong>Total 4xx Errors</strong></td><td>{smart_format(login_data.get('seg1_4xx', 'N/A'))}</td><td>{smart_format(login_data.get('seg2_4xx', 'N/A'))}</td><td>{smart_format(login_data.get('overall_4xx', 'N/A'))}</td></tr>"
    html += f"<tr><td><strong>Total 5xx Errors</strong></td><td>{smart_format(login_data.get('seg1_5xx', 'N/A'))}</td><td>{smart_format(login_data.get('seg2_5xx', 'N/A'))}</td><td>{smart_format(login_data.get('overall_5xx', 'N/A'))}</td></tr>"
    html += f"<tr><td><strong>Total Failures</strong></td><td>{smart_format(login_data.get('seg1_failures', 'N/A'))}</td><td>{smart_format(login_data.get('seg2_failures', 'N/A'))}</td><td>{smart_format(login_data.get('overall_failures', 'N/A'))}</td></tr>"
    html += "</tbody></table>"
    
    failure_map_str = json.dumps(login_data.get('status_code_failure_map', {}))
    html += f"<h4>Status Code Failure Map: <code>{failure_map_str}</code></h4>"
    html += "</section>"
    return html

def format_dashboard(dashboard_data):
    """Generates HTML for the Dashboard Load Times section."""
    if not dashboard_data:
        return "<section><h2>2. 📊 Overall Dashboard Load Times</h2><p>No dashboard data found.</p></section>"
        
    html = "<section><h2>2. 📊 Overall Dashboard Load Times</h2><table>"
    html += "<thead><tr><th>Dashboard Name</th><th>Segment</th><th>Time Range</th><th>Total Loads</th><th>Success Rate</th><th>Avg. Load (ms)</th><th>P95 Load (ms)</th><th>Errors 4xx</th><th>Errors 5xx</th><th>Errors Conn</th></tr></thead>"
    html += "<tbody>"
    
    for item in dashboard_data:
        html += "<tr>"
        html += f"<td>{item.get('dashboard_name', 'N/A')}</td>"
        html += f"<td>{smart_format(item.get('segment', 'N/A'), 0)}</td>"
        html += f"<td>{item.get('time_range', 'N/A')}</td>"
        html += f"<td>{smart_format(item.get('total_loads', 'N/A'), 0)}</td>"
        html += f"<td>{smart_format(item.get('success_rate', 'N/A'))}%</td>"
        html += f"<td>{smart_format(item.get('avg_load_ms', 'N/A'))}</td>"
        html += f"<td>{smart_format(item.get('p95_load_ms', 'N/A'))}</td>"
        html += f"<td>{smart_format(item.get('errors_4xx', 'N/A'), 0)}</td>"
        html += f"<td>{smart_format(item.get('errors_5xx', 'N/A'), 0)}</td>"
        html += f"<td>{smart_format(item.get('errors_conn', 'N/A'), 0)}</td>"
        html += "</tr>"
        
    html += "</tbody></table></section>"
    return html

def create_panel_chart(panel_data):
    """Generates a Plotly horizontal bar chart for the top 10 slowest panels."""
    try:
        if not panel_data:
            return ""
            
        df = pd.DataFrame(panel_data)
        # Handle potential non-numeric data gracefully
        df['avg_load_ms'] = pd.to_numeric(df['avg_load_ms'], errors='coerce')
        df = df.dropna(subset=['avg_load_ms'])
        
        if df.empty:
            return ""
            
        # Get top 10
        df_top10 = df.nlargest(10, 'avg_load_ms').sort_values('avg_load_ms', ascending=True)
        
        fig = px.bar(df_top10, 
                     x='avg_load_ms', 
                     y='panel_name_id', 
                     orientation='h', 
                     title='Top 10 Slowest Panels by Avg. Load Time',
                     labels={'avg_load_ms': 'Avg. Load (ms)', 'panel_name_id': 'Panel'},
                     text='avg_load_ms',
                     color_discrete_sequence=['#9f7aea']) # Soft purple
        
        fig.update_traces(texttemplate='%{text:.2f} ms', textposition='outside')
        fig.update_layout(height=500, yaxis_title_text='')
        
        return f"<div class='chart-full'>{pio.to_html(fig, full_html=False)}</div>"
    except Exception as e:
        return f"<p>⚠️ Could not generate panel chart: {e}</p>"

def format_panels(panel_data):
    """Generates HTML for the Panel Performance section."""
    if not panel_data:
        return "<section><h2>3. 🖼️ Panel Performance Breakdown</h2><p>No panel data found.</p></section>"

    html = "<section><h2>3. 🖼️ Panel Performance Breakdown</h2>"
    
    html += create_panel_chart(panel_data)
    
    html += "<table><thead><tr><th>Dashboard</th><th>Panel Name (ID)</th><th>Attempts</th><th>Failed</th><th>Avg. Load (ms)</th><th>P95 Load (ms)</th><th>Avg. Contrib. %</th><th>Success Rate</th><th>4xx</th><th>5xx</th><th>Conn</th></tr></thead>"
    html += "<tbody>"
    
    for item in panel_data:
        html += "<tr>"
        html += f"<td>{item.get('dashboard', 'N/A')}</td>"
        html += f"<td>{item.get('panel_name_id', 'N/A')}</td>"
        html += f"<td>{smart_format(item.get('total_attempts', 'N/A'), 0)}</td>"
        html += f"<td>{smart_format(item.get('failed_attempts', 'N/A'), 0)}</td>"
        html += f"<td>{smart_format(item.get('avg_load_ms', 'N/A'))}</td>"
        html += f"<td>{smart_format(item.get('p95_load_ms', 'N/A'))}</td>"
        html += f"<td>{smart_format(item.get('avg_contribution_pct', 'N/A'))}%</td>"
        html += f"<td>{smart_format(item.get('success_rate', 'N/A'))}%</td>"
        html += f"<td>{smart_format(item.get('errors_4xx', 'N/A'), 0)}</td>"
        html += f"<td>{smart_format(item.get('errors_5xx', 'N/A'), 0)}</td>"
        html += f"<td>{smart_format(item.get('errors_conn', 'N/A'), 0)}</td>"
        html += "</tr>"
        
    html += "</tbody></table></section>"
    return html

# ===============================
# MAIN EXECUTION
# ===============================

def main():
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    
    try:
        conn = sqlite3.connect(DB_PATH)
        # Updated query to include start_time, end_time, and vus
        query = f"""
            SELECT 
                test_id, test_name, status, duration, o11y_sources,
                start_time, end_time, vus,
                summarised, 
                Metrics_Login, 
                Overall_Dashboard_Load_Times, 
                Panel_Performance_Breakdown
            FROM {TABLE_NAME}
            WHERE 
                summarised IS NOT NULL AND summarised != '' AND
                Metrics_Login IS NOT NULL AND Metrics_Login != ''
        """
        df = pd.read_sql_query(query, conn)
        conn.close()
        print(f"✅ Loaded {len(df)} valid test runs from database '{DB_PATH}'.\n")
        
        if len(df) == 0:
            print("⚠️ No test runs with K6 metrics found. Exiting.")
            return

    except Exception as e:
        print(f"❌ Error loading data from SQLite DB '{DB_PATH}', table '{TABLE_NAME}'. Error: {e}")
        return

    for _, row in df.iterrows():
        test_id = row["test_id"]
        print(f"Processing report for: {test_id}")
        
        try:
            # Safely parse JSON data using the helper function
            summary_raw = get_json_str(row.get("summarised"), default_fallback="{}")
            login_raw = get_json_str(row.get("Metrics_Login"), default_fallback="{}")
            dashboard_raw = get_json_str(row.get("Overall_Dashboard_Load_Times"), default_fallback="[]")
            panel_raw = get_json_str(row.get("Panel_Performance_Breakdown"), default_fallback="[]")
            
            summary_data = json.loads(summary_raw)
            login_data = json.loads(login_raw)
            dashboard_data = json.loads(dashboard_raw)
            panel_data = json.loads(panel_raw)

            # Generate HTML sections
            html_summary = format_summary(row) # Updated to only need the row for DB fields
            html_login = format_login(login_data)
            html_dashboard = format_dashboard(dashboard_data)
            html_panel = format_panels(panel_data)
            
            final_content = html_summary + html_login + html_dashboard + html_panel
            final_html = wrap_html_template(final_content, test_id)

            file_name = f"k6_report_{test_id}.html"
            file_path = os.path.join(OUTPUT_DIR, file_name)
            
            with open(file_path, "w", encoding="utf-8") as f:
                f.write(final_html)
            print(f"📝 Generated: {file_path}")

        except Exception as e:
            print(f"⚠️ Failed to generate report for {test_id}. Error: {e}")

    print(f"\n✅ All HTML reports generated successfully in '{OUTPUT_DIR}' folder.")

if __name__ == "__main__":
    main()
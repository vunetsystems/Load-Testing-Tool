import os
import sqlite3
import pandas as pd
import json
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
    """Safely handles data of various types and returns a string for json.loads."""
    try:
        s = str(value).strip()
    except:
        return default_fallback
        
    if s.lower() in ('none', 'nan', 'true', 'false', '0', '1', ''):
        return default_fallback
    return s

def wrap_html_template(content, test_id, test_name):
    """Wraps content in a minimal HTML5 page with compressed CSS."""
    title_text = f"K6 Report: {test_name}"
    header_text = f"📊 K6 Report: {test_name}"

    # Minified CSS - removed redundant styles and compressed
    return f"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>{title_text}</title>
<style>
body{{font-family:system-ui,-apple-system,sans-serif;margin:1rem auto;max-width:1100px;background:#f8fafc;color:#333;line-height:1.5}}
h1,h2,h3,h4{{color:#1a365d;margin:.5em 0}}
section{{background:#fff;padding:1rem;border-radius:8px;box-shadow:0 1px 3px rgba(0,0,0,.1);margin-bottom:1rem}}
table{{width:100%;border-collapse:collapse;margin-top:.5rem;font-size:.9em}}
th,td{{border:1px solid #ddd;padding:6px 8px;text-align:left}}
th{{background:#1a365d;color:#fff}}
tr:nth-child(even){{background:#f9f9f9}}
tr:hover{{background:#e6f0ff}}
code{{background:#edf2f7;padding:2px 4px;border-radius:3px;font-size:.9em}}
.desc{{background:#f0f8ff;border-left:3px solid #1a365d;padding:8px 12px;margin:.5rem 0;font-size:.9em}}
.chart-container{{display:flex;flex-wrap:wrap;gap:1rem;margin-top:1rem}}
.chart{{flex:1;min-width:400px;box-shadow:0 1px 3px rgba(0,0,0,.1);border-radius:8px;background:#fff;padding:.5rem}}
.chart-full{{width:100%;box-shadow:0 1px 3px rgba(0,0,0,.1);border-radius:8px;background:#fff;margin-top:1rem;padding:.5rem}}
</style>
</head>
<body>
<h1>{header_text}</h1>
{content}
</body>
</html>"""

def format_summary(row):
    """Generates compact HTML for Test Configuration."""
    html = "<section><h2>Test Configuration</h2><table>"
    html += f"<tr><td><strong>Test ID</strong></td><td><code>{row.get('test_id', 'N/A')}</code></td></tr>"
    html += f"<tr><td><strong>Test Name</strong></td><td>{row.get('test_name', 'N/A')}</td></tr>"
    html += f"<tr><td><strong>Status</strong></td><td>{row.get('status', 'N/A')}</td></tr>"
    html += f"<tr><td><strong>VUS</strong></td><td>{smart_format(row.get('vus', 'N/A'), 0)}</td></tr>"
    html += f"<tr><td><strong>Start</strong></td><td>{row.get('start_time', 'N/A')}</td></tr>"
    html += f"<tr><td><strong>End</strong></td><td>{row.get('end_time', 'N/A')}</td></tr>"
    html += f"<tr><td><strong>Duration</strong></td><td>{row.get('duration', 'N/A')}</td></tr>"
    html += f"<tr><td><strong>Sources</strong></td><td>{row.get('o11y_sources', 'N/A')}</td></tr>"
    html += "</table></section>"
    return html

def create_login_charts(login_data):
    """Generates compact Plotly charts with reduced config."""
    try:
        df = pd.DataFrame({
            'Segment': ['Sequential', 'Parallel'],
            'Avg RT (ms)': [
                float(login_data.get('seg1_avg_rt', 0)), 
                float(login_data.get('seg2_avg_rt', 0))
            ],
            'Success %': [
                float(login_data.get('seg1_success_rate', 0)), 
                float(login_data.get('seg2_success_rate', 0))
            ]
        })

        # Minimal chart config
        config = {'displayModeBar': False, 'responsive': True}
        
        fig_rt = px.bar(df, x='Segment', y='Avg RT (ms)', 
                        title='Login Avg Response Time',
                        color_discrete_sequence=['#1a365d'])
        fig_rt.update_traces(texttemplate='%{y:.1f}', textposition='outside')
        fig_rt.update_layout(height=300, margin=dict(l=20,r=20,t=40,b=20))

        fig_sr = px.bar(df, x='Segment', y='Success %', 
                        title='Login Success Rate',
                        range_y=[0, 100],
                        color_discrete_sequence=['#38a169'])
        fig_sr.update_traces(texttemplate='%{y:.1f}%', textposition='outside')
        fig_sr.update_layout(height=300, margin=dict(l=20,r=20,t=40,b=20))

        # Use include_plotlyjs='cdn' only once, then False for others
        chart_html = "<div class='chart-container'>"
        chart_html += f"<div class='chart'>{pio.to_html(fig_rt, full_html=False, include_plotlyjs='cdn', config=config)}</div>"
        chart_html += f"<div class='chart'>{pio.to_html(fig_sr, full_html=False, include_plotlyjs=False, config=config)}</div>"
        chart_html += "</div>"
        return chart_html
    except Exception as e:
        return f"<p>⚠️ Chart error: {e}</p>"

def format_login(login_data):
    """Generates compact HTML for Login Performance."""
    html = "<section><h2>1. 🔑 Login Performance</h2>"
    html += "<p class='desc'><strong>Sequential:</strong><br>What it does: Runs login first, then dashboard tests, one after the other.<br>Purpose / Intent: Simulate realistic user behavior where users log in before accessing dashboards. Captures metrics under a controlled, sequential load.<br><br><strong>Parallel:</strong><br>What it does: Runs login and dashboard tests simultaneously.<br>Purpose / Intent: Simulate concurrent user actions or stress testing scenarios. Captures metrics under parallel load to identify potential bottlenecks.</p>"
    
    html += create_login_charts(login_data)
    
    html += "<table><thead><tr><th>Metric</th><th>Sequential Run</th><th>Parallel Run</th><th>Overall</th></tr></thead><tbody>"
    html += f"<tr><td>Attempts</td><td>{smart_format(login_data.get('seg1_attempts', 'N/A'), 0)}</td><td>{smart_format(login_data.get('seg2_attempts', 'N/A'), 0)}</td><td>{smart_format(login_data.get('overall_attempts', 'N/A'), 0)}</td></tr>"
    html += f"<tr><td>Success %</td><td>{smart_format(login_data.get('seg1_success_rate', 'N/A'))}</td><td>{smart_format(login_data.get('seg2_success_rate', 'N/A'))}</td><td>{smart_format(login_data.get('overall_success_rate', 'N/A'))}</td></tr>"
    html += f"<tr><td>Avg RT (ms)</td><td>{smart_format(login_data.get('seg1_avg_rt', 'N/A'))}</td><td>{smart_format(login_data.get('seg2_avg_rt', 'N/A'))}</td><td>{smart_format(login_data.get('overall_avg_rt', 'N/A'))}</td></tr>"
    html += f"<tr><td>P95 RT (ms)</td><td>{smart_format(login_data.get('seg1_p95_rt', 'N/A'))}</td><td>{smart_format(login_data.get('seg2_p95_rt', 'N/A'))}</td><td>{smart_format(login_data.get('overall_p95_rt', 'N/A'))}</td></tr>"
    html += f"<tr><td>P99 RT (ms)</td><td>{smart_format(login_data.get('seg1_p99_rt', 'N/A'))}</td><td>{smart_format(login_data.get('seg2_p99_rt', 'N/A'))}</td><td>{smart_format(login_data.get('overall_p99_rt', 'N/A'))}</td></tr>"
    html += f"<tr><td>4xx Errors</td><td>{smart_format(login_data.get('seg1_4xx', 'N/A'), 0)}</td><td>{smart_format(login_data.get('seg2_4xx', 'N/A'), 0)}</td><td>{smart_format(login_data.get('overall_4xx', 'N/A'), 0)}</td></tr>"
    html += f"<tr><td>5xx Errors</td><td>{smart_format(login_data.get('seg1_5xx', 'N/A'), 0)}</td><td>{smart_format(login_data.get('seg2_5xx', 'N/A'), 0)}</td><td>{smart_format(login_data.get('overall_5xx', 'N/A'), 0)}</td></tr>"
    html += f"<tr><td>Failures</td><td>{smart_format(login_data.get('seg1_failures', 'N/A'), 0)}</td><td>{smart_format(login_data.get('seg2_failures', 'N/A'), 0)}</td><td>{smart_format(login_data.get('overall_failures', 'N/A'), 0)}</td></tr>"
    html += "</tbody></table>"
    
    # Compact failure map display
    failure_map = login_data.get('status_code_failure_map', {})
    if failure_map:
        html += f"<p style='margin-top:.5rem;font-size:.85em'><strong>Status Failures:</strong> <code>{json.dumps(failure_map)}</code></p>"
    html += "</section>"
    return html

def format_dashboard(dashboard_data):
    """Generates compact HTML for Dashboard metrics."""
    html = "<section><h2>2. 📊 Dashboard Loading Time</h2>"
    html += "<p class='desc'>End-to-end dashboard loading performance metrics</p>"

    if not dashboard_data:
        html += "<p>No data</p></section>"
        return html
        
    html += "<table><thead><tr><th>Dashboard</th><th>Seg</th><th>Range</th><th>Loads</th><th>Success%</th><th>Avg(ms)</th><th>P95(ms)</th><th>4xx</th><th>5xx</th><th>Conn</th></tr></thead><tbody>"
    
    for item in dashboard_data:
        html += f"<tr><td>{item.get('dashboard_name', 'N/A')}</td>"
        html += f"<td>{smart_format(item.get('segment', 'N/A'), 0)}</td>"
        html += f"<td>{item.get('time_range', 'N/A')}</td>"
        html += f"<td>{smart_format(item.get('total_loads', 'N/A'), 0)}</td>"
        html += f"<td>{smart_format(item.get('success_rate', 'N/A'))}</td>"
        html += f"<td>{smart_format(item.get('avg_load_ms', 'N/A'))}</td>"
        html += f"<td>{smart_format(item.get('p95_load_ms', 'N/A'))}</td>"
        html += f"<td>{smart_format(item.get('errors_4xx', 'N/A'), 0)}</td>"
        html += f"<td>{smart_format(item.get('errors_5xx', 'N/A'), 0)}</td>"
        html += f"<td>{smart_format(item.get('errors_conn', 'N/A'), 0)}</td></tr>"
        
    html += "</tbody></table></section>"
    return html

def create_panel_chart(panel_data):
    """Generates compact panel chart."""
    try:
        if not panel_data:
            return ""
            
        df = pd.DataFrame(panel_data)
        df['avg_load_ms'] = pd.to_numeric(df['avg_load_ms'], errors='coerce')
        df = df.dropna(subset=['avg_load_ms'])
        
        if df.empty:
            return ""
            
        df_top10 = df.nlargest(10, 'avg_load_ms').sort_values('avg_load_ms', ascending=True)
        
        config = {'displayModeBar': False, 'responsive': True}
        
        fig = px.bar(df_top10, x='avg_load_ms', y='panel_name_id', orientation='h', 
                     title='Top 10 Slowest Panels',
                     labels={'avg_load_ms': 'Avg (ms)', 'panel_name_id': 'Panel'},
                     color_discrete_sequence=['#9f7aea'])
        
        fig.update_traces(texttemplate='%{x:.1f}', textposition='outside')
        fig.update_layout(height=400, margin=dict(l=20,r=20,t=40,b=20), yaxis_title='')
        
        return f"<div class='chart-full'>{pio.to_html(fig, full_html=False, include_plotlyjs=False, config=config)}</div>"
    except Exception as e:
        return f"<p>⚠️ Chart error: {e}</p>"

def format_panels(panel_data):
    """Generates compact HTML for Panel Performance."""
    html = "<section><h2>3. 🖼️ Panel Performance</h2>"
    html += "<p class='desc'>Individual panel query execution metrics</p>"

    if not panel_data:
        html += "<p>No data</p></section>"
        return html

    html += create_panel_chart(panel_data)
    
    html += "<table><thead><tr><th>Dashboard</th><th>Panel</th><th>Attempts</th><th>Failed</th><th>Avg(ms)</th><th>P95(ms)</th><th>Contrib%</th><th>Success%</th><th>4xx</th><th>5xx</th><th>Conn</th></tr></thead><tbody>"
    
    for item in panel_data:
        html += f"<tr><td>{item.get('dashboard', 'N/A')}</td>"
        html += f"<td>{item.get('panel_name_id', 'N/A')}</td>"
        html += f"<td>{smart_format(item.get('total_attempts', 'N/A'), 0)}</td>"
        html += f"<td>{smart_format(item.get('failed_attempts', 'N/A'), 0)}</td>"
        html += f"<td>{smart_format(item.get('avg_load_ms', 'N/A'))}</td>"
        html += f"<td>{smart_format(item.get('p95_load_ms', 'N/A'))}</td>"
        html += f"<td>{smart_format(item.get('avg_contribution_pct', 'N/A'))}</td>"
        html += f"<td>{smart_format(item.get('success_rate', 'N/A'))}</td>"
        html += f"<td>{smart_format(item.get('errors_4xx', 'N/A'), 0)}</td>"
        html += f"<td>{smart_format(item.get('errors_5xx', 'N/A'), 0)}</td>"
        html += f"<td>{smart_format(item.get('errors_conn', 'N/A'), 0)}</td></tr>"
        
    html += "</tbody></table></section>"
    return html

# ===============================
# MAIN EXECUTION
# ===============================

def main():
    os.makedirs(OUTPUT_DIR, exist_ok=True)
    
    try:
        conn = sqlite3.connect(DB_PATH)
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
                Metrics_Login IS NOT NULL AND Metrics_Login != '' AND
                report_generated = 0
        """
        df = pd.read_sql_query(query, conn)
        conn.close()
        print(f"✅ Loaded {len(df)} valid test runs from database '{DB_PATH}'.\n")

        if len(df) == 0:
            print("⚠️ No unprocessed test runs with K6 metrics found. Exiting.")
            return

    except Exception as e:
        print(f"❌ Error loading data from SQLite DB '{DB_PATH}', table '{TABLE_NAME}'. Error: {e}")
        return

    for _, row in df.iterrows():
        test_id = row["test_id"]
        test_name = row.get("test_name", f"Report_{test_id}")
        print(f"Processing report for: {test_name}")
        
        try:
            summary_raw = get_json_str(row.get("summarised"), default_fallback="{}")
            login_raw = get_json_str(row.get("Metrics_Login"), default_fallback="{}")
            dashboard_raw = get_json_str(row.get("Overall_Dashboard_Load_Times"), default_fallback="[]")
            panel_raw = get_json_str(row.get("Panel_Performance_Breakdown"), default_fallback="[]")
            
            summary_data = json.loads(summary_raw)
            login_data = json.loads(login_raw)
            dashboard_data = json.loads(dashboard_raw)
            panel_data = json.loads(panel_raw)

            # Generate HTML sections
            html_summary = format_summary(row)
            html_login = format_login(login_data)
            html_dashboard = format_dashboard(dashboard_data)
            html_panel = format_panels(panel_data)
            
            final_content = html_summary + html_login + html_dashboard + html_panel
            final_html = wrap_html_template(final_content, test_id, test_name)

            safe_test_name = "".join(c if c.isalnum() or c in (' ', '_', '-') else '_' for c in test_name).strip()
            file_name = f"k6_report_{safe_test_name}_{test_id[:8]}.html"
            file_path = os.path.join(OUTPUT_DIR, file_name)
            
            with open(file_path, "w", encoding="utf-8") as f:
                f.write(final_html)
            print(f"📝 Generated: {file_path}")

            # Update report_generated to 1
            update_conn = sqlite3.connect(DB_PATH)
            update_conn.execute("UPDATE k6_runs SET report_generated = 1 WHERE test_id = ?", (test_id,))
            update_conn.commit()
            update_conn.close()

        except Exception as e:
            print(f"⚠️ Failed to generate report for {test_name} ({test_id}). Error: {e}")

    print(f"\n✅ All HTML reports generated successfully in '{OUTPUT_DIR}' folder.")

if __name__ == "__main__":
    main()
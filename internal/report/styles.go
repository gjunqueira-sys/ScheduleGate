package report

// cssStyles defines the shared CSS for report generation.
// Theme: Dark Mode "ScheduleGate" with gradients.
const cssStyles = `
:root {
	--bg-dark: #0f172a;
	--bg-card: #1e293b;
	--text-main: #f1f5f9;
	--text-muted: #94a3b8;
	--accent-primary: #3b82f6; 
	--accent-gradient: linear-gradient(135deg, #3b82f6 0%, #8b5cf6 100%);
	--border-color: #334155;
	--success: #10b981;
	--warning: #f59e0b;
	--danger: #ef4444;
}

body {
	font-family: 'Inter', system-ui, -apple-system, sans-serif;
	background-color: var(--bg-dark);
	color: var(--text-main);
	margin: 0;
	padding: 2rem;
	line-height: 1.6;
	-webkit-font-smoothing: antialiased;
}

.container {
	max-width: 1100px;
	margin: 0 auto;
}

/* Header */
header {
	margin-bottom: 3rem;
	border-bottom: 1px solid var(--border-color);
	padding-bottom: 1.5rem;
	display: flex;
	justify-content: space-between;
	align-items: flex-end;
}

h1 {
	font-size: 2.5rem;
	font-weight: 800;
	margin: 0;
	background: var(--accent-gradient);
	-webkit-background-clip: text;
	-webkit-text-fill-color: transparent;
	display: inline-block;
}

.ascii-logo {
	font-family: 'Courier New', monospace;
	font-size: 10px;
	line-height: 10px;
	background: linear-gradient(90deg, #06b6d4 0%, #3b82f6 100%);
	-webkit-background-clip: text;
	-webkit-text-fill-color: transparent;
	font-weight: bold;
	white-space: pre;
	margin: 0;
	padding-bottom: 1rem;
	overflow-x: hidden; /* Prevent scrollbar if it barely fits */
}

.meta-info {
	text-align: right;
	font-size: 0.9rem;
	color: var(--text-muted);
}

.meta-item {
	margin-bottom: 0.25rem;
}

.meta-label {
	font-weight: 600;
	color: var(--text-main);
}

/* Cards */
.card {
	background: var(--bg-card);
	border: 1px solid var(--border-color);
	border-radius: 12px;
	padding: 1.5rem;
	margin-bottom: 1.5rem;
	box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.3);
}

.section-title {
	font-size: 1.2rem;
	font-weight: 700;
	margin-bottom: 1.5rem;
	color: var(--text-main);
	text-transform: uppercase;
	letter-spacing: 0.05em;
	border-left: 4px solid var(--accent-primary);
	padding-left: 10px;
}

/* Score Badge */
.score-container {
	text-align: center;
	padding: 2rem;
}

.score-value {
	font-size: 5rem;
	font-weight: 900;
	line-height: 1;
	margin-bottom: 0.5rem;
}

.score-subtitle {
	color: var(--text-muted);
	font-size: 1.1rem;
	font-weight: 500;
}

.text-success { color: var(--success); }
.text-warning { color: var(--warning); }
.text-danger { color: var(--danger); }

/* Grid Layouts */
.grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem; }
.grid-3 { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 1.5rem; }

@media (max-width: 768px) {
	.grid-2, .grid-3 { grid-template-columns: 1fr; }
	header { flex-direction: column; align-items: flex-start; gap: 1rem; }
	.meta-info { text-align: left; }
}

/* Metrics List (Assess) */
.metric-row {
	display: flex;
	justify-content: space-between;
	align-items: center;
	padding: 1rem;
	border-bottom: 1px solid var(--border-color);
	transition: background 0.2s;
}
.metric-row:last-child { border-bottom: none; }
.metric-row:hover { background: rgba(255,255,255,0.03); }

.badge {
	padding: 0.25rem 0.75rem;
	border-radius: 99px;
	font-size: 0.75rem;
	font-weight: 700;
	text-transform: uppercase;
}
.badge-pass { background: rgba(16, 185, 129, 0.2); color: var(--success); border: 1px solid var(--success); }
.badge-fail { background: rgba(239, 68, 68, 0.2); color: var(--danger); border: 1px solid var(--danger); }

/* Progress Bars (Compare) */
.progress-container {
	margin-bottom: 1.5rem;
}
.progress-label {
	display: flex;
	justify-content: space-between;
	margin-bottom: 0.5rem;
	font-weight: 600;
}
.progress-bg {
	height: 12px;
	background: #334155;
	border-radius: 6px;
	overflow: hidden;
}
.progress-fill {
	height: 100%;
	border-radius: 6px;
	transition: width 0.5s ease-out;
}

/* Tables */
table {
	width: 100%;
	border-collapse: collapse;
	font-size: 0.9rem;
}
th {
	text-align: left;
	padding: 0.75rem 1rem;
	color: var(--text-muted);
	font-weight: 600;
	border-bottom: 1px solid var(--border-color);
}
td {
	padding: 0.75rem 1rem;
	border-bottom: 1px solid rgba(255,255,255,0.05);
}
tr:last-child td { border-bottom: none; }

.cell-num { font-family: monospace; font-weight: 600; }
`

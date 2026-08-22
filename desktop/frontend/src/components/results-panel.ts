import type {
  AssessResponse, CompareResponse, ValidateResponse, CheckPatternsResponse, DCMAMetricDef,
} from '../types'

const METRICS: DCMAMetricDef[] = [
  { id: 1, name: 'Logic', selected: true },
  { id: 2, name: 'Leads', selected: true },
  { id: 3, name: 'Lags', selected: true },
  { id: 4, name: 'Relationship Types', selected: true },
  { id: 5, name: 'Hard Constraints', selected: true },
  { id: 6, name: 'High Float', selected: true },
  { id: 7, name: 'Negative Float', selected: true },
  { id: 8, name: 'High Duration', selected: true },
  { id: 9, name: 'Invalid Dates', selected: true },
  { id: 10, name: 'Resources', selected: true },
  { id: 11, name: 'Missed Tasks', selected: true },
  { id: 12, name: 'Critical Path Test', selected: true },
  { id: 13, name: 'CPLI', selected: true },
  { id: 14, name: 'BEI', selected: true },
]

function scoreColor(score: number): string {
  if (score >= 80) return 'sc-high'
  if (score >= 50) return 'sc-mid'
  return 'sc-low'
}

function statusBadge(passing: boolean): string {
  return passing
    ? '<span class="badge badge-pass">PASS</span>'
    : '<span class="badge badge-na">N/A</span>' // will be overridden for fail
}

export function renderAssessResults(resp: AssessResponse) {
  const el = document.getElementById('results-content')!
  if (!resp.success) {
    el.innerHTML = `<div class="result-error">${resp.error || 'Assessment failed'}</div>`
    return
  }

  let html = '<div class="result-score">'

  const sc = scoreColor(resp.overallScore)
  html += `<div class="score-circle ${sc}">${resp.overallScore.toFixed(1)}%</div>`
  html += `<div class="score-label">${resp.passed}/${resp.total} Metrics Passed</div>`

  if (resp.scheduleName) {
    html += `<div class="score-meta">${resp.scheduleName} · ${resp.dataDate} · ${resp.taskCount} tasks</div>`
  }
  html += '</div>'

  html += '<div class="metric-list">'
  for (const m of resp.metrics || []) {
    if (m.notApplicable) {
      html += `
        <div class="metric-row">
          <div class="metric-name">${m.id}. ${m.name}</div>
          <div class="metric-value na">N/A</div>
        </div>`
    } else {
      const cls = m.passing ? 'pass' : 'fail'
      html += `
        <div class="metric-row">
          <div class="metric-name">${m.id}. ${m.name}</div>
          <div class="metric-value ${cls}">${m.value.toFixed(1)}%</div>
          <span class="badge ${m.passing ? 'badge-pass' : 'badge-fail'}">${m.passing ? 'PASS' : 'FAIL'}</span>
        </div>`
    }
  }
  html += '</div>'

  if (resp.outputFiles?.length) {
    html += '<div class="output-files">'
    html += '<div class="out-title">Reports</div>'
    for (const f of resp.outputFiles) {
      html += `<div class="out-file">📄 ${f.split('/').pop() || f}</div>`
    }
    html += '</div>'
  }

  if (resp.errors?.length) {
    html += '<div class="result-errors">'
    for (const e of resp.errors) {
      html += `<div class="err-item">⚠ ${e}</div>`
    }
    html += '</div>'
  }

  el.innerHTML = html
}

export function renderCompareResults(resp: CompareResponse) {
  const el = document.getElementById('results-content')!
  if (!resp.success) {
    el.innerHTML = `<div class="result-error">${resp.error || 'Comparison failed'}</div>`
    return
  }

  let html = '<div class="result-score">'
  const sc = scoreColor(resp.overallScore)
  html += `<div class="score-circle ${sc}">${resp.overallScore.toFixed(1)}%</div>`
  html += '<div class="score-label">Overall Score</div>'
  html += '</div>'

  html += '<div class="pillar-grid">'
  html += pillarBar('Stability (40%)', resp.pillarAScore, 40)
  html += pillarBar('Reliability (30%)', resp.pillarBScore, 30)
  html += pillarBar('Churn (30%)', resp.pillarCScore, 30)
  html += '</div>'

  html += '<div class="stats-grid">'
  html += statCard('Total Tasks', String(resp.totalTasks))
  html += statCard('New', String(resp.newTasks))
  html += statCard('Deleted', String(resp.deletedTasks))
  html += statCard('Modified', String(resp.modifiedTasks))
  html += statCard('Unchanged', String(resp.unchangedTasks))
  html += statCard('Ghost Tasks', String(resp.ghostTasksCount))
  html += statCard('Duration Inflated', resp.durationInflatedPct.toFixed(1) + '%')
  html += '</div>'

  if (resp.outputFiles?.length) {
    html += '<div class="output-files">'
    for (const f of resp.outputFiles) {
      html += `<div class="out-file">📄 ${f.split('/').pop() || f}</div>`
    }
    html += '</div>'
  }

  el.innerHTML = html
}

export function renderValidateResults(resp: ValidateResponse) {
  const el = document.getElementById('results-content')!
  if (!resp.success) {
    el.innerHTML = `<div class="result-error">Validation errored</div>`
    return
  }
  const passing = resp.missing === 0
  const sc = passing ? 'sc-high' : 'sc-low'
  let html = '<div class="result-score">'
  html += `<div class="score-circle ${sc}"><span style="font-size:24px">${passing ? 'PASSED' : 'FAILED'}</span></div>`
  html += `<div class="score-label">${resp.found} found · ${resp.missing} missing · ${resp.extra} extra</div>`
  html += '</div>'
  el.innerHTML = html
}

export function renderPatternResults(resp: CheckPatternsResponse) {
  const el = document.getElementById('results-content')!
  if (!resp.success) {
    el.innerHTML = `<div class="result-error">Pattern check failed</div>`
    return
  }
  const results = resp.results || []
  const passed = results.filter(r => r.passing).length
  const total = results.length
  const allPassing = passed === total
  const sc = allPassing ? 'sc-high' : 'sc-low'

  let html = '<div class="result-score">'
  html += `<div class="score-circle ${sc}"><span style="font-size:22px">${allPassing ? 'COMPLIANT' : 'NON-COMPLIANT'}</span></div>`
  html += `<div class="score-label">${passed}/${total} Rules Passed · ${resp.taskCount} tasks</div>`
  html += '</div>'

  html += '<div class="metric-list">'
  for (const r of results) {
    const cls = r.passing ? 'pass' : 'fail'
    html += `
      <div class="metric-row">
        <div class="metric-name">${r.name}</div>
        <div class="metric-value ${cls}">${r.count}</div>
        <span class="badge ${r.passing ? 'badge-pass' : 'badge-fail'}">${r.passing ? 'PASS' : 'FAIL'}</span>
      </div>`
  }
  html += '</div>'

  el.innerHTML = html
}

function pillarBar(label: string, score: number, max: number): string {
  const pct = (score / max) * 100
  const color = pct >= 70 ? 'var(--success)' : pct >= 40 ? 'var(--warning)' : 'var(--danger)'
  return `
    <div class="pillar-item">
      <div class="pillar-header"><span>${label}</span><span>${score.toFixed(1)}/${max}</span></div>
      <div class="pillar-track"><div class="pillar-fill" style="width:${pct}%;background:${color}"></div></div>
    </div>`
}

function statCard(label: string, value: string): string {
  return `<div class="stat-card"><div class="stat-val">${value}</div><div class="stat-lbl">${label}</div></div>`
}

import type { AssessRequest, CompareRequest, ValidateRequest, CheckPatternsRequest, DCMAMetricDef } from '../types'
import { OpenScheduleFile, OpenRulesFile, SaveFileDialog } from '../../bindings/github.com/gjunqueira/schedule-benchmark-cli/desktop/services/fileservice.js'

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

export type Command = 'assess' | 'compare' | 'validate' | 'patterns'
export type RunCallback = (params: {
  command: Command
  assess?: AssessRequest
  compare?: CompareRequest
  validate?: ValidateRequest
  patterns?: CheckPatternsRequest
}) => void

interface StringRecord { [id: string]: string }
interface BoolRecord { [id: string]: boolean }
interface MetricsState { metrics: boolean[] }

interface FormState {
  assess: {
    file: string
    html: string
    csv: string
    excel: string
    statusDate: string
    pctFormat: string
    dateLocale: string
    verbose: boolean
    debug: boolean
    customer: string
    project: string
    metrics: boolean[]
  }
  compare: {
    prev: string
    curr: string
    html: string
    csv: string
    pctFormat: string
    dateLocale: string
    detailed: boolean
    customer: string
    project: string
  }
  validate: {
    file: string
  }
  patterns: {
    file: string
    rules: string
    pctFormat: string
    dateLocale: string
    detailed: boolean
  }
}

function defaultState(): FormState {
  return {
    assess: {
      file: '', html: '', csv: '', excel: '',
      statusDate: '', pctFormat: '', dateLocale: 'US',
      verbose: false, debug: false,
      customer: '', project: '',
      metrics: METRICS.map(() => true),
    },
    compare: {
      prev: '', curr: '', html: '', csv: '',
      pctFormat: '', dateLocale: 'US',
      detailed: false, customer: '', project: '',
    },
    validate: {
      file: '',
    },
    patterns: {
      file: '', rules: '',
      pctFormat: '', dateLocale: 'US',
      detailed: false,
    },
  }
}

export class Sidebar {
  private command: Command = 'assess'
  private onRun: RunCallback
  private state: FormState

  constructor(onRun: RunCallback) {
    this.onRun = onRun
    this.state = defaultState()
    this.initCommandTabs()
    this.renderAssessForm()
  }

  private initCommandTabs() {
    const tabs = document.querySelectorAll<HTMLElement>('#cmd-tabs .cmd-tab')
    tabs.forEach(tab => {
      tab.addEventListener('click', () => {
        tabs.forEach(t => t.classList.remove('active'))
        tab.classList.add('active')
        this.command = tab.dataset.cmd as Command
        this.renderForm()
      })
    })
    document.getElementById('btn-run')!.addEventListener('click', () => this.handleRun())
    document.getElementById('btn-stop')!.addEventListener('click', () => {
      window.dispatchEvent(new CustomEvent('run-stop'))
    })
  }

  private snapshot() {
    const s = this.state
    switch (this.command) {
      case 'assess': {
        s.assess.file = getVal('assess-file')
        s.assess.html = getVal('assess-html')
        s.assess.csv = getVal('assess-csv')
        s.assess.excel = getVal('assess-excel')
        s.assess.statusDate = getVal('assess-status-date')
        s.assess.pctFormat = getVal('assess-pct-format')
        s.assess.dateLocale = getVal('assess-date-locale')
        s.assess.verbose = getChecked('assess-verbose')
        s.assess.debug = getChecked('assess-debug')
        s.assess.customer = getVal('assess-customer')
        s.assess.project = getVal('assess-project')
        const cbs = document.querySelectorAll<HTMLInputElement>('#metrics-grid input[type=checkbox]')
        s.assess.metrics = Array.from(cbs).map(c => c.checked)
        break
      }
      case 'compare': {
        s.compare.prev = getVal('cmp-prev')
        s.compare.curr = getVal('cmp-curr')
        s.compare.html = getVal('cmp-html')
        s.compare.csv = getVal('cmp-csv')
        s.compare.pctFormat = getVal('cmp-pct-format')
        s.compare.dateLocale = getVal('cmp-date-locale')
        s.compare.detailed = getChecked('cmp-detailed')
        s.compare.customer = getVal('cmp-customer')
        s.compare.project = getVal('cmp-project')
        break
      }
      case 'validate': {
        s.validate.file = getVal('val-file')
        break
      }
      case 'patterns': {
        s.patterns.file = getVal('pat-file')
        s.patterns.rules = getVal('pat-rules')
        s.patterns.pctFormat = getVal('pat-pct-format')
        s.patterns.dateLocale = getVal('pat-date-locale')
        s.patterns.detailed = getChecked('pat-detailed')
        break
      }
    }
  }

  private renderForm() {
    this.snapshot()
    switch (this.command) {
      case 'assess': this.renderAssessForm(); break
      case 'compare': this.renderCompareForm(); break
      case 'validate': this.renderValidateForm(); break
      case 'patterns': this.renderPatternsForm(); break
    }
  }

  private renderAssessForm() {
    const s = this.state.assess
    const html = `
      <div class="field">
        <label>Schedule File</label>
        <div class="file-row">
          <input type="text" id="assess-file" placeholder="No file selected" readonly value="${esc(s.file)}">
          <button class="btn btn-sm" id="btn-assess-file">Browse</button>
        </div>
      </div>
      <div class="field">
        <label>Metrics</label>
        <div class="metrics-grid" id="metrics-grid">
          ${METRICS.map((m, i) => `
            <label class="metric-check" data-id="${m.id}">
              <input type="checkbox" ${s.metrics[i] ? 'checked' : ''}> ${m.name}
            </label>
          `).join('')}
        </div>
        <button class="btn-link" id="btn-metrics-all">Select All</button>
        <button class="btn-link" id="btn-metrics-none">Deselect All</button>
      </div>
      <div class="section-title-sm">Output</div>
      <div class="field">
        <label>HTML Report</label>
        <div class="file-row">
          <input type="text" id="assess-html" placeholder="report.html" value="${esc(s.html)}">
          <button class="btn btn-sm" id="btn-assess-html">Save</button>
        </div>
      </div>
      <div class="field">
        <label>CSV Database</label>
        <div class="file-row">
          <input type="text" id="assess-csv" placeholder="schedule_db.csv" value="${esc(s.csv)}">
          <button class="btn btn-sm" id="btn-assess-csv">Save</button>
        </div>
      </div>
      <div class="field">
        <label>Excel Exceptions</label>
        <div class="file-row">
          <input type="text" id="assess-excel" placeholder="exceptions.xlsx" value="${esc(s.excel)}">
          <button class="btn btn-sm" id="btn-assess-excel">Save</button>
        </div>
      </div>
      <div class="section-title-sm">Advanced</div>
      <div class="field">
        <label>Status Date</label>
        <input type="text" id="assess-status-date" placeholder="YYYY-MM-DD (default: today)" value="${esc(s.statusDate)}">
      </div>
      <div class="field">
        <label>Percent Format</label>
        <select id="assess-pct-format">
          <option value="">0-100 (MS Project, default)</option>
          <option value="fraction" ${s.pctFormat === 'fraction' ? 'selected' : ''}>Fraction 0.0-1.0 (Primavera, etc.)</option>
        </select>
      </div>
      <div class="field">
        <label>Date Locale</label>
        <select id="assess-date-locale">
          <option value="US" ${s.dateLocale === 'US' ? 'selected' : ''}>US (MM/DD/YYYY)</option>
          <option value="EU" ${s.dateLocale === 'EU' ? 'selected' : ''}>EU (DD/MM/YYYY)</option>
        </select>
      </div>
      <div class="field-row">
        <label class="toggle-label">
          <input type="checkbox" id="assess-verbose" ${s.verbose ? 'checked' : ''}> Verbose
        </label>
        <label class="toggle-label">
          <input type="checkbox" id="assess-debug" ${s.debug ? 'checked' : ''}> Debug Logic
        </label>
      </div>
      <div class="field">
        <label>Customer</label>
        <input type="text" id="assess-customer" placeholder="Customer name" value="${esc(s.customer)}">
      </div>
      <div class="field">
        <label>Project</label>
        <input type="text" id="assess-project" placeholder="Project ID" value="${esc(s.project)}">
      </div>
    `
    document.getElementById('flags-panel')!.innerHTML = html

    document.getElementById('btn-assess-file')?.addEventListener('click', async () => {
      try {
        const path = await OpenScheduleFile()
        if (path) { (document.getElementById('assess-file') as HTMLInputElement).value = path; this.state.assess.file = path }
      } catch (e) {}
    })
    document.getElementById('btn-assess-html')?.addEventListener('click', async () => {
      try {
        const path = await SaveFileDialog('Save HTML Report', 'report.html')
        if (path) { (document.getElementById('assess-html') as HTMLInputElement).value = path; this.state.assess.html = path }
      } catch (e) {}
    })
    document.getElementById('btn-assess-csv')?.addEventListener('click', async () => {
      try {
        const path = await SaveFileDialog('Save CSV Database', 'schedule_db.csv')
        if (path) { (document.getElementById('assess-csv') as HTMLInputElement).value = path; this.state.assess.csv = path }
      } catch (e) {}
    })
    document.getElementById('btn-assess-excel')?.addEventListener('click', async () => {
      try {
        const path = await SaveFileDialog('Save Excel Exceptions', 'exceptions.xlsx')
        if (path) { (document.getElementById('assess-excel') as HTMLInputElement).value = path; this.state.assess.excel = path }
      } catch (e) {}
    })
    document.getElementById('btn-metrics-all')?.addEventListener('click', () => {
      document.querySelectorAll<HTMLInputElement>('#metrics-grid input[type=checkbox]').forEach(c => c.checked = true)
    })
    document.getElementById('btn-metrics-none')?.addEventListener('click', () => {
      document.querySelectorAll<HTMLInputElement>('#metrics-grid input[type=checkbox]').forEach(c => c.checked = false)
    })
  }

  private renderCompareForm() {
    const s = this.state.compare
    const html = `
      <div class="field">
        <label>Previous Schedule</label>
        <div class="file-row">
          <input type="text" id="cmp-prev" placeholder="No file selected" readonly value="${esc(s.prev)}">
          <button class="btn btn-sm" id="btn-cmp-prev">Browse</button>
        </div>
      </div>
      <div class="field">
        <label>Current Schedule</label>
        <div class="file-row">
          <input type="text" id="cmp-curr" placeholder="No file selected" readonly value="${esc(s.curr)}">
          <button class="btn btn-sm" id="btn-cmp-curr">Browse</button>
        </div>
      </div>
      <div class="section-title-sm">Output</div>
      <div class="field">
        <label>HTML Report</label>
        <div class="file-row">
          <input type="text" id="cmp-html" placeholder="comparison.html" value="${esc(s.html)}">
          <button class="btn btn-sm" id="btn-cmp-html">Save</button>
        </div>
      </div>
      <div class="field">
        <label>CSV Database</label>
        <div class="file-row">
          <input type="text" id="cmp-csv" placeholder="compare_db.csv" value="${esc(s.csv)}">
          <button class="btn btn-sm" id="btn-cmp-csv">Save</button>
        </div>
      </div>
      <div class="section-title-sm">Advanced</div>
      <div class="field">
        <label>Percent Format</label>
        <select id="cmp-pct-format">
          <option value="">0-100 (MS Project, default)</option>
          <option value="fraction" ${s.pctFormat === 'fraction' ? 'selected' : ''}>Fraction 0.0-1.0 (Primavera, etc.)</option>
        </select>
      </div>
      <div class="field">
        <label>Date Locale</label>
        <select id="cmp-date-locale">
          <option value="US" ${s.dateLocale === 'US' ? 'selected' : ''}>US (MM/DD/YYYY)</option>
          <option value="EU" ${s.dateLocale === 'EU' ? 'selected' : ''}>EU (DD/MM/YYYY)</option>
        </select>
      </div>
      <div class="field-row">
        <label class="toggle-label">
          <input type="checkbox" id="cmp-detailed" ${s.detailed ? 'checked' : ''}> Detailed Analysis
        </label>
      </div>
      <div class="field">
        <label>Customer</label>
        <input type="text" id="cmp-customer" placeholder="Customer name" value="${esc(s.customer)}">
      </div>
      <div class="field">
        <label>Project</label>
        <input type="text" id="cmp-project" placeholder="Project ID" value="${esc(s.project)}">
      </div>
    `
    document.getElementById('flags-panel')!.innerHTML = html

    document.getElementById('btn-cmp-prev')?.addEventListener('click', async () => {
      try {
        const path = await OpenScheduleFile()
        if (path) { (document.getElementById('cmp-prev') as HTMLInputElement).value = path; this.state.compare.prev = path }
      } catch (e) {}
    })
    document.getElementById('btn-cmp-curr')?.addEventListener('click', async () => {
      try {
        const path = await OpenScheduleFile()
        if (path) { (document.getElementById('cmp-curr') as HTMLInputElement).value = path; this.state.compare.curr = path }
      } catch (e) {}
    })
    document.getElementById('btn-cmp-html')?.addEventListener('click', async () => {
      try {
        const path = await SaveFileDialog('Save HTML Report', 'comparison.html')
        if (path) { (document.getElementById('cmp-html') as HTMLInputElement).value = path; this.state.compare.html = path }
      } catch (e) {}
    })
    document.getElementById('btn-cmp-csv')?.addEventListener('click', async () => {
      try {
        const path = await SaveFileDialog('Save CSV Database', 'compare_db.csv')
        if (path) { (document.getElementById('cmp-csv') as HTMLInputElement).value = path; this.state.compare.csv = path }
      } catch (e) {}
    })
  }

  private renderValidateForm() {
    const s = this.state.validate
    const html = `
      <div class="field">
        <label>Schedule File</label>
        <div class="file-row">
          <input type="text" id="val-file" placeholder="No file selected" readonly value="${esc(s.file)}">
          <button class="btn btn-sm" id="btn-val-file">Browse</button>
        </div>
      </div>
    `
    document.getElementById('flags-panel')!.innerHTML = html

    document.getElementById('btn-val-file')?.addEventListener('click', async () => {
      try {
        const path = await OpenScheduleFile()
        if (path) { (document.getElementById('val-file') as HTMLInputElement).value = path; this.state.validate.file = path }
      } catch (e) {}
    })
  }

  private renderPatternsForm() {
    const s = this.state.patterns
    const html = `
      <div class="field">
        <label>Schedule File</label>
        <div class="file-row">
          <input type="text" id="pat-file" placeholder="No file selected" readonly value="${esc(s.file)}">
          <button class="btn btn-sm" id="btn-pat-file">Browse</button>
        </div>
      </div>
      <div class="field">
        <label>Rules File (YAML)</label>
        <div class="file-row">
          <input type="text" id="pat-rules" placeholder="No rules selected" readonly value="${esc(s.rules)}">
          <button class="btn btn-sm" id="btn-pat-rules">Browse</button>
        </div>
      </div>
      <div class="section-title-sm">Advanced</div>
      <div class="field">
        <label>Percent Format</label>
        <select id="pat-pct-format">
          <option value="">0-100 (MS Project, default)</option>
          <option value="fraction" ${s.pctFormat === 'fraction' ? 'selected' : ''}>Fraction 0.0-1.0 (Primavera, etc.)</option>
        </select>
      </div>
      <div class="field">
        <label>Date Locale</label>
        <select id="pat-date-locale">
          <option value="US" ${s.dateLocale === 'US' ? 'selected' : ''}>US (MM/DD/YYYY)</option>
          <option value="EU" ${s.dateLocale === 'EU' ? 'selected' : ''}>EU (DD/MM/YYYY)</option>
        </select>
      </div>
      <div class="field-row">
        <label class="toggle-label">
          <input type="checkbox" id="pat-detailed" ${s.detailed ? 'checked' : ''}> Detailed Matches
        </label>
      </div>
    `
    document.getElementById('flags-panel')!.innerHTML = html

    document.getElementById('btn-pat-file')?.addEventListener('click', async () => {
      try {
        const path = await OpenScheduleFile()
        if (path) { (document.getElementById('pat-file') as HTMLInputElement).value = path; this.state.patterns.file = path }
      } catch (e) {}
    })
    document.getElementById('btn-pat-rules')?.addEventListener('click', async () => {
      try {
        const path = await OpenRulesFile()
        if (path) { (document.getElementById('pat-rules') as HTMLInputElement).value = path; this.state.patterns.rules = path }
      } catch (e) {}
    })
  }

  private handleRun() {
    this.snapshot()
    const params: Parameters<RunCallback>[0] = { command: this.command }

    switch (this.command) {
      case 'assess': {
        const s = this.state.assess
        params.assess = {
          filePath: s.file,
          metrics: METRICS.map((m, i) => s.metrics[i] ? m.id : 0).filter(id => id !== 0),
          htmlOutput: s.html,
          csvOutput: s.csv,
          exceptionsReport: s.excel,
          customer: s.customer,
          project: s.project,
          verbose: s.verbose,
          statusDate: s.statusDate,
          debugLogic: s.debug,
          pctFormat: s.pctFormat,
          dateLocale: s.dateLocale,
        }
        break
      }
      case 'compare': {
        const s = this.state.compare
        params.compare = {
          previousFile: s.prev,
          currentFile: s.curr,
          htmlOutput: s.html,
          csvOutput: s.csv,
          detailed: s.detailed,
          customer: s.customer,
          project: s.project,
          pctFormat: s.pctFormat,
          dateLocale: s.dateLocale,
        }
        break
      }
      case 'validate': {
        const s = this.state.validate
        params.validate = {
          filePath: s.file,
          htmlOutput: '',
          csvOutput: '',
        }
        break
      }
      case 'patterns': {
        const s = this.state.patterns
        params.patterns = {
          filePath: s.file,
          rulesFile: s.rules,
          htmlOutput: '',
          csvOutput: '',
          detailed: s.detailed,
          pctFormat: s.pctFormat,
          dateLocale: s.dateLocale,
        }
        break
      }
    }
    this.onRun(params)
  }
}

function esc(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

function getVal(id: string): string {
  return (document.getElementById(id) as HTMLInputElement)?.value || ''
}

function getChecked(id: string): boolean {
  return (document.getElementById(id) as HTMLInputElement)?.checked || false
}

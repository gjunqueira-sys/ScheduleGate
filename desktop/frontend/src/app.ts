import type {
  AssessRequest, CompareRequest, ValidateRequest, CheckPatternsRequest,
} from './types'
import { TerminalPanel } from './components/terminal-panel'
import { Sidebar, type Command } from './components/sidebar'
import {
  renderAssessResults, renderCompareResults,
  renderValidateResults, renderPatternResults,
} from './components/results-panel'
import { Events } from '@wailsio/runtime'
import { Run as RunAssess } from '../bindings/github.com/gjunqueira/schedule-benchmark-cli/desktop/services/assessservice.js'
import { Run as RunCompare } from '../bindings/github.com/gjunqueira/schedule-benchmark-cli/desktop/services/compareservice.js'
import { Run as RunValidate } from '../bindings/github.com/gjunqueira/schedule-benchmark-cli/desktop/services/validateservice.js'
import { Run as RunPatterns } from '../bindings/github.com/gjunqueira/schedule-benchmark-cli/desktop/services/checkpatternsservice.js'

let terminal: TerminalPanel
let isRunning = false

function init() {
  terminal = new TerminalPanel('terminal')
  terminal.writeln('\x1b[1;34mScheduleGate\x1b[0m  \x1b[90mDCMA 14-Point Assessment Tool\x1b[0m')
  terminal.writeln('\x1b[90mReady. Select a command and files to begin.\x1b[0m\n')

  new Sidebar((params) => runCommand(params))

  window.addEventListener('run-stop', () => { isRunning = false })

  document.getElementById('btn-clear')?.addEventListener('click', () => {
    terminal.clear()
    terminal.writeln('\x1b[90mTerminal cleared.\x1b[0m')
  })
}

async function runCommand(params: {
  command: Command
  assess?: AssessRequest
  compare?: CompareRequest
  validate?: ValidateRequest
  patterns?: CheckPatternsRequest
}) {
  if (isRunning) return
  isRunning = true

  const runBtn = document.getElementById('btn-run')!
  const stopBtn = document.getElementById('btn-stop')!
  runBtn.classList.add('hidden')
  stopBtn.classList.remove('hidden')

  document.getElementById('results-content')!.innerHTML =
    '<div class="result-loading">Running... <span class="loading-dots"></span></div>'

  const offTermOutput = Events.On('term-output', (ev: { data: unknown }) => {
    if (!isRunning) return
    terminal.write(String(ev.data))
  })
  const offTermLoading = Events.On('term-loading', (ev: { data: unknown }) => {
    if (!isRunning) return
    if (ev.data) terminal.startLoading()
    else terminal.stopLoading()
  })

  try {
    switch (params.command) {
      case 'assess': {
        if (!params.assess) break
        const resp = await RunAssess(params.assess)
        if (resp && isRunning) renderAssessResults(resp)
        break
      }
      case 'compare': {
        if (!params.compare) break
        const resp = await RunCompare(params.compare)
        if (resp && isRunning) renderCompareResults(resp)
        break
      }
      case 'validate': {
        if (!params.validate) break
        const resp = await RunValidate(params.validate)
        if (resp && isRunning) renderValidateResults(resp)
        break
      }
      case 'patterns': {
        if (!params.patterns) break
        const resp = await RunPatterns(params.patterns)
        if (resp && isRunning) renderPatternResults(resp)
        break
      }
    }
  } catch (err) {
    if (isRunning) terminal.writeln(`\r\n\x1b[31mError: ${String(err)}\x1b[0m`)
  } finally {
    isRunning = false
    runBtn.classList.remove('hidden')
    stopBtn.classList.add('hidden')
    terminal.stopLoading()
    offTermOutput()
    offTermLoading()
  }
}

window.addEventListener('DOMContentLoaded', () => init())

import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'

export class TerminalPanel {
  private term: Terminal
  private fitAddon: FitAddon
  private container: HTMLElement
  private dotsTimer: ReturnType<typeof setInterval> | null = null
  private readonly DOTS = '·  ·  ·'
  private readonly DOTS_WIDTH = 30
  private isDisposed = false

  constructor(containerId: string) {
    this.container = document.getElementById(containerId)!

    this.term = new Terminal({
      theme: {
        background: '#080c14',
        foreground: '#e4e8f0',
        cursor: '#4f8fff',
        cursorAccent: '#080c14',
        selectionBackground: '#4f8fff44',
        black: '#0f1520',
        red: '#f24e5e',
        green: '#2dd47e',
        yellow: '#f5a623',
        blue: '#4f8fff',
        magenta: '#9b6dff',
        cyan: '#00c8dc',
        white: '#e4e8f0',
        brightBlack: '#5a6678',
        brightRed: '#ff6b77',
        brightGreen: '#4fffa2',
        brightYellow: '#ffc043',
        brightBlue: '#7aafff',
        brightMagenta: '#b88dff',
        brightCyan: '#33e5f5',
        brightWhite: '#ffffff',
      },
      fontSize: 13,
      fontFamily: '"JetBrains Mono", "Fira Code", "SF Mono", Menlo, monospace',
      cursorBlink: true,
      cursorStyle: 'bar',
      convertEol: true,
      disableStdin: true,
      allowProposedApi: true,
      scrollback: 10000,
      smoothScrollDuration: 50,
    })

    this.fitAddon = new FitAddon()
    this.term.loadAddon(this.fitAddon)
    this.term.open(this.container)

    const observer = new ResizeObserver(() => {
      if (!this.isDisposed) {
        try { this.fitAddon.fit() } catch (_) { /* ignore */ }
      }
    })
    observer.observe(this.container)
  }

  write(text: string) {
    if (this.isDisposed) return
    // Stop any active loading spinner before writing real output so the
    // spinner's periodic carriage-return redraw can't overwrite content.
    if (this.dotsTimer) this.stopLoading()
    try { this.term.write(text) } catch (_) { /* ignore */ }
  }

  writeln(text: string) {
    this.write(text + '\n')
  }

  clear() {
    if (this.isDisposed) return
    try { this.term.clear() } catch (_) { /* ignore */ }
  }

  startLoading() {
    this.stopLoading()
    let pos = 0
    let dir = 1
    this.dotsTimer = setInterval(() => {
      if (this.isDisposed) { this.stopLoading(); return }
      pos += dir
      if (pos >= this.DOTS_WIDTH - 7) dir = -1
      if (pos <= 0) dir = 1
      try {
        this.term.write(`\r\x1b[K\x1b[90m${' '.repeat(pos)}${this.DOTS}\x1b[0m`)
      } catch (_) { /* ignore */ }
    }, 100)
  }

  stopLoading() {
    if (this.dotsTimer) {
      clearInterval(this.dotsTimer)
      this.dotsTimer = null
      if (!this.isDisposed) {
        try { this.term.write('\r\x1b[K') } catch (_) { /* ignore */ }
      }
    }
  }

  dispose() {
    this.isDisposed = true
    this.stopLoading()
    try { this.term.dispose() } catch (_) { /* ignore */ }
  }
}

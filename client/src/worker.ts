/** Local OpenPort worker over WebSocket. The UI never imports J2534. */
import type { DetectedAdapter, DidStream, ScanResult } from './types'

const wsUrl = import.meta.env.VITE_WORKER_WS ?? 'ws://127.0.0.1:8765'

type WorkerReply<T> = { id: string; ok: boolean; result?: T; error?: string }

export class WorkerClient {
  private ws: WebSocket | null = null
  private pending = new Map<string, (reply: WorkerReply<unknown>) => void>()
  private seq = 0
  private opening: Promise<void> | null = null

  async connectSocket(): Promise<void> {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) return
    if (this.opening) return this.opening
    this.ws = null
    this.opening = new Promise<void>((resolve, reject) => {
      const ws = new WebSocket(wsUrl)
      const fail = () => {
        if (this.ws === ws) this.ws = null
        reject(new Error(`Diagnostic worker is not running at ${wsUrl}. Start it with make worker.`))
      }
      ws.onopen = () => {
        this.ws = ws
        resolve()
      }
      ws.onerror = fail
      ws.onclose = () => {
        if (this.ws === ws) this.ws = null
      }
      ws.onmessage = (ev) => {
        const reply = JSON.parse(String(ev.data)) as WorkerReply<unknown>
        const wait = this.pending.get(String(reply.id))
        if (wait) {
          this.pending.delete(String(reply.id))
          wait(reply)
        }
      }
    }).finally(() => {
      this.opening = null
    })
    return this.opening
  }

  private async request<T>(cmd: string, extra: Record<string, unknown> = {}, timeoutMs = 15000): Promise<T> {
    await this.connectSocket()
    const id = String(++this.seq)
    const reply = await new Promise<WorkerReply<T>>((resolve, reject) => {
      const timer = window.setTimeout(() => reject(new Error('worker timeout')), timeoutMs)
      this.pending.set(id, (r) => {
        window.clearTimeout(timer)
        resolve(r as WorkerReply<T>)
      })
      this.ws?.send(JSON.stringify({ id, cmd, ...extra }))
    })
    if (!reply.ok) throw new Error(reply.error ?? 'worker error')
    return reply.result as T
  }

  status() {
    return this.request<{ connected: boolean; adapter: string | null }>('status')
  }

  detect() {
    return this.request<{ devices: DetectedAdapter[]; recommended: string; j2534_lib: string | null }>('detect')
  }

  connectAdapter(adapter: string) {
    return this.request<{ adapter: string; connected: boolean; library: string | null }>('connect', { adapter })
  }

  identify(vin?: string) {
    return this.request<{ vin: string; profile: string; make: string; model: string; year: number; coverage?: { id: string; depth: string; gaps: string[] } }>('identify', vin ? { vin } : {}, 25000)
  }

  scan(hints?: { vin?: string; make?: string; model?: string; year?: number }) {
    return this.request<ScanResult>('scan', { ...hints }, 45000)
  }

  streamDids(seconds = 6) {
    return this.request<DidStream>('stream_dids', { seconds }, 25000)
  }
}

export const worker = new WorkerClient()

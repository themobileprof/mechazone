import type { DidStream, ScanResult } from './types'

const wsUrl = import.meta.env.VITE_WORKER_WS ?? 'ws://127.0.0.1:8765'

type WorkerReply<T> = { id: string; ok: boolean; result?: T; error?: string }

export class WorkerClient {
  private ws: WebSocket | null = null
  private pending = new Map<string, (reply: WorkerReply<unknown>) => void>()
  private seq = 0

  async connectSocket(): Promise<void> {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) return
    await new Promise<void>((resolve, reject) => {
      const ws = new WebSocket(wsUrl)
      ws.onopen = () => {
        this.ws = ws
        resolve()
      }
      ws.onerror = () => reject(new Error('worker websocket failed'))
      ws.onmessage = (ev) => {
        const reply = JSON.parse(String(ev.data)) as WorkerReply<unknown>
        const wait = this.pending.get(String(reply.id))
        if (wait) {
          this.pending.delete(String(reply.id))
          wait(reply)
        }
      }
    })
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

  connectAdapter(adapter: 'mock' | 'openport2_rev_e') {
    return this.request<{ adapter: string; connected: boolean; library: string | null }>('connect', { adapter })
  }

  identify() {
    return this.request<{ vin: string }>('identify')
  }

  scan() {
    return this.request<ScanResult>('scan', {}, 45000)
  }

  streamDids(seconds = 6) {
    return this.request<DidStream>('stream_dids', { seconds }, 25000)
  }
}

export const worker = new WorkerClient()

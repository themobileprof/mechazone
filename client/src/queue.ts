/** Offline JSON queue for session/closeout/customer writes. Imported reports are not queued. */
import type { QueuedJob } from './types'

const KEY = 'mechazone.sync_queue'

function read(): QueuedJob[] {
  try {
    return JSON.parse(localStorage.getItem(KEY) ?? '[]') as QueuedJob[]
  } catch {
    return []
  }
}

function write(jobs: QueuedJob[]) {
  localStorage.setItem(KEY, JSON.stringify(jobs))
}

export function enqueue(job: Omit<QueuedJob, 'id' | 'created_at'>) {
  const jobs = read().filter((j) => !(job.kind === 'customer' && j.kind === 'customer' && j.path === job.path))
  jobs.push({
    ...job,
    id: crypto.randomUUID(),
    created_at: new Date().toISOString(),
  })
  write(jobs)
}

export function pendingCount(): number {
  return read().length
}

export async function flushQueue(): Promise<number> {
  const jobs = read()
  const remain: QueuedJob[] = []
  let sent = 0
  for (const job of jobs) {
    try {
      const res = await fetch(job.path, {
        method: job.method || 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(job.body),
      })
      if (!res.ok) throw new Error(String(res.status))
      sent += 1
    } catch {
      remain.push(job)
    }
  }
  write(remain)
  return sent
}

from __future__ import annotations

import asyncio
import logging
import os

from mechazone_worker.ipc import serve


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")
    host = os.environ.get("WORKER_WS_HOST", "127.0.0.1")
    port = int(os.environ.get("WORKER_WS_PORT", "8765"))
    asyncio.run(serve(host, port))


if __name__ == "__main__":
    main()

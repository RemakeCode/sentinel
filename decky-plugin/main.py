import asyncio
import logging
import os
import signal


class Plugin:
    def __init__(self):
        self.process = None
        self.stdout_task = None
        self.stderr_task = None

    async def _main(self):
        bin_path = os.path.join(os.environ['DECKY_PLUGIN_DIR'], "bin", "sentinel-dev")
        try:
            os.chmod(bin_path, 0o755)
        except PermissionError:
            logging.warning(f"Could not set executable permission on {bin_path}")

        logging.info(f"Starting binary at {bin_path}")

        try:
            self.process = await asyncio.create_subprocess_exec(
                bin_path, '--decky',
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
                start_new_session=True
            )
            self.stderr_task = asyncio.create_task(self._drain_stderr())
            self.stdout_task = asyncio.create_task(self._drain_stdout())
        except Exception as e:
            logging.error(f"Failed to start binary: {e}")
            await self._stop_process()

    async def _drain_stdout(self):
        if self.process is None or self.process.stdout is None:
            return

        while await self.process.stdout.readline():
            pass

    async def _drain_stderr(self):
        if self.process is None or self.process.stderr is None:
            return

        while True:
            line = await self.process.stderr.readline()
            if not line:
                break
            logging.error(f"[sentinel-dev stderr] {line.decode(errors='replace').rstrip()}")

    async def _stop_process(self):
        if self.stdout_task is not None:
            self.stdout_task.cancel()
            self.stdout_task = None

        if self.stderr_task is not None:
            self.stderr_task.cancel()
            self.stderr_task = None

        if self.process is None:
            return

        logging.info("Stopping binary...")
        try:
            pgid = os.getpgid(self.process.pid)
            os.killpg(pgid, signal.SIGTERM)

            try:
                await asyncio.wait_for(self.process.wait(), timeout=5)
            except asyncio.TimeoutError:
                logging.warning("Process group did not terminate gracefully, sending SIGKILL")
                os.killpg(pgid, signal.SIGKILL)
                await asyncio.wait_for(self.process.wait(), timeout=3)
        except ProcessLookupError:
            pass
        except Exception as e:
            logging.error(f"Error during cleanup: {e}")
        finally:
            self.process = None

    async def _unload(self):
        await self._stop_process()

import asyncio
import logging
import os
import signal


class Plugin:
    def __init__(self):
        self.process = None

    async def _main(self):
        bin_path = os.path.join(os.environ['DECKY_PLUGIN_DIR'], "bin", "sentinel-decky")
        try:
            os.chmod(bin_path, 0o755)
        except PermissionError:
            logging.warning(f"Could not set executable permission on {bin_path}")

        logging.info(f"Starting binary at {bin_path}")

        try:
            self.process = await asyncio.create_subprocess_exec(
                bin_path, '--decky',
                start_new_session=True
            )
        except Exception as e:
            logging.error(f"Failed to start binary: {e}")
            await self._stop_process()

    async def _stop_process(self):
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

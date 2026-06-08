import asyncio
import logging
import os
import signal

AUTH_TOKEN_PREFIX = "SENTINEL_DECKY_AUTH_TOKEN="
AUTH_TOKEN_TIMEOUT_SECONDS = 10


def parse_auth_token_line(line: str):
    line = line.strip()
    if line.startswith(AUTH_TOKEN_PREFIX):
        token = line[len(AUTH_TOKEN_PREFIX):].strip()
        return token if token else None
    return None


class Plugin:
    def __init__(self):
        self.process = None
        self.auth_token = None
        self.stdout_task = None

    async def _main(self):
        bin_path = os.path.join(os.environ['DECKY_PLUGIN_DIR'], "bin", "sentinel-dev")
        os.chmod(bin_path, 0o755)

        logging.info(f"Starting binary at {bin_path}")

        try:
            self.process = await asyncio.create_subprocess_exec(
                bin_path, '--decky',
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.DEVNULL,
                start_new_session=True
            )
            self.auth_token = await self._capture_auth_token()
            self.stdout_task = asyncio.create_task(self._drain_stdout())
        except Exception as e:
            logging.error(f"Failed to start binary: {e}")
            await self._stop_process()

    async def _capture_auth_token(self):
        if self.process is None or self.process.stdout is None:
            raise RuntimeError("Sentinel process stdout is unavailable")

        while True:
            line = await asyncio.wait_for(self.process.stdout.readline(), timeout=AUTH_TOKEN_TIMEOUT_SECONDS)
            if not line:
                raise RuntimeError("Sentinel process exited before emitting Decky auth token")

            token = parse_auth_token_line(line.decode(errors='replace'))
            if token:
                logging.info("Captured Decky auth token from Sentinel startup")
                return token

    async def _drain_stdout(self):
        if self.process is None or self.process.stdout is None:
            return

        while await self.process.stdout.readline():
            pass

    async def get_decky_auth_token(self):
        if not self.auth_token:
            raise RuntimeError("Decky auth token is not available")
        return self.auth_token

    async def _stop_process(self):
        if self.stdout_task is not None:
            self.stdout_task.cancel()
            self.stdout_task = None

        if self.process is None:
            self.auth_token = None
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
            self.auth_token = None

    async def _unload(self):
        await self._stop_process()

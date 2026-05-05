import asyncio
import logging
import os
import signal


class Plugin:
    def __init__(self):
        self.process = None

    async def _main(self):
        bin_path = os.path.join(os.environ['DECKY_PLUGIN_DIR'], "bin", "sentinel")
        os.chmod(bin_path, 0o755)

        # 1. Prevent duplicates: Kill any lingering instances from hard crashes
        # where _unload() was bypassed.
        try:
            pkill_proc = await asyncio.create_subprocess_exec(
                "pkill", "-f", bin_path,
                stdout=asyncio.subprocess.DEVNULL,
                stderr=asyncio.subprocess.DEVNULL
            )
            await pkill_proc.wait()
        except Exception as e:
            logging.debug(f"Pre-launch cleanup skipped or failed: {e}")

        logging.info(f"Starting binary at {bin_path}")

        try:
            self.process = await asyncio.create_subprocess_exec(
                bin_path, '--decky',
                stdout=asyncio.subprocess.DEVNULL,
                stderr=asyncio.subprocess.DEVNULL,
                start_new_session=True  # Becomes the leader of a new process group
            )
        except Exception as e:
            logging.error(f"Failed to start binary: {e}")

    async def _unload(self):
        if self.process is not None:
            logging.info("Stopping binary...")
            try:
                # 2. Prevent zombies: Get the Process Group ID (PGID) and kill the whole group.
                # This ensures any child processes spawned by your binary are also terminated.
                pgid = os.getpgid(self.process.pid)
                os.killpg(pgid, signal.SIGTERM)

                try:
                    await asyncio.wait_for(self.process.wait(), timeout=5)
                except asyncio.TimeoutError:
                    logging.warning("Process group did not terminate gracefully, sending SIGKILL")
                    os.killpg(pgid, signal.SIGKILL)
                    await asyncio.wait_for(self.process.wait(), timeout=3)

            except ProcessLookupError:
                # Process has already exited on its own
                logging.info("Binary process already exited.")
            except Exception as e:
                logging.error(f"Error during cleanup: {e}")
            finally:
                # 3. Always clear the reference
                self.process = None

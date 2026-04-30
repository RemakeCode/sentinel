import logging
import os
import subprocess


class Plugin:
    def __init__(self):
        self.process = None

    async def _main(self):
        # Define the path to your binary inside the plugin folder
        bin_path = os.path.join(os.environ['DECKY_PLUGIN_DIR'], "bin", "sentinel")

        # Ensure the binary is executable
        os.chmod(bin_path, 0o755)
        logging.info(f"Starting binary at {bin_path}")

        try:
            # Use Popen so it runs in the background
            self.process = subprocess.Popen(
                [bin_path, '--decky'],
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True
            )
        except Exception as e:
            logging.error(f"Failed to start binary: {e}")

    async def _unload(self):
        # CRITICAL: Kill the binary when Decky unloads the plugin
        if self.process:
            logging.info("Stopping binary...")
            self.process.terminate()
            self.process.wait()

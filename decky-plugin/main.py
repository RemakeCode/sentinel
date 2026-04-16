import os
import subprocess

class Plugin:
    async def _main(self):
        # We assume the Go binary is placed in a 'bin' folder
        # inside the plugin directory during the build/zip process
        bin_path = os.path.join(os.path.dirname(__file__), "bin", "my-app-binary")

        # Launching the Wails binary in headless mode
        subprocess.Popen([bin_path, "-headless"])

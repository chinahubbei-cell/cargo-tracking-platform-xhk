import http.server
import socketserver
import os

PORT = 8000
DIRECTORY = "dist"

class SPAHandler(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=DIRECTORY, **kwargs)

    def do_GET(self):
        # 尝试寻找文件
        path = self.translate_path(self.path)
        if not os.path.exists(path) or os.path.isdir(path) and not os.path.exists(os.path.join(path, "index.html")):
            # 如果文件不存在，或者路径是目录且没有index.html，则返回根目录的 index.html
            self.path = "/"
        super().do_GET()

print(f"Starting SPA Server at http://127.0.0.1:{PORT}")
print(f"Serving directory: {DIRECTORY}")

with socketserver.TCPServer(("", PORT), SPAHandler) as httpd:
    httpd.serve_forever()

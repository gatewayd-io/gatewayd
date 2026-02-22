"""Mock prediction API for gatewayd-plugin-sql-ids-ips CI tests.

Exposes POST /predict endpoint that classifies SQL queries as malicious
or safe using simple pattern matching (no ML model needed).

Returns {"confidence": 0.99} for known SQLi patterns,
        {"confidence": 0.01} for normal queries.
"""

import json
import re
import sys
from http.server import HTTPServer, BaseHTTPRequestHandler

SQLI_PATTERNS = [
    r"\bOR\s+1\s*=\s*1\b",
    r"\bOR\s+'[^']*'\s*=\s*'[^']*'",
    r"\bUNION\s+(ALL\s+)?SELECT\b",
    r";\s*(DROP|DELETE|UPDATE|INSERT|ALTER|CREATE)\b",
    r"\bSELECT\b.*\bFROM\b.*\binformation_schema\b",
    r"--\s*$",
    r"\b(SLEEP|BENCHMARK|WAITFOR)\s*\(",
    r"\bpg_sleep\s*\(",
    r"\bEXTRACTVALUE\s*\(",
    r"\bCONCAT\s*\(.*0x",
    r"'\s*(AND|OR)\s+\d+\s*=\s*\d+",
    r"\bHAVING\s+\d+\s*=\s*\d+",
    r"\bGROUP\s+BY\s+.+--",
    r"\bORDER\s+BY\s+\d+--",
]

COMPILED_PATTERNS = [re.compile(p, re.IGNORECASE) for p in SQLI_PATTERNS]


def is_sqli(query: str) -> bool:
    return any(p.search(query) for p in COMPILED_PATTERNS)


class PredictHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        if self.path != "/predict":
            self.send_error(404)
            return

        length = int(self.headers.get("Content-Length", 0))
        body = json.loads(self.rfile.read(length)) if length else {}
        query = body.get("query", "")
        confidence = 0.99 if is_sqli(query) else 0.01

        response = json.dumps({"confidence": confidence})
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(response.encode())

    def log_message(self, fmt, *args):
        print(f"[mock-sqli-api] {fmt % args}", file=sys.stderr)


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8000
    server = HTTPServer(("0.0.0.0", port), PredictHandler)
    print(f"[mock-sqli-api] Listening on port {port}", file=sys.stderr)
    server.serve_forever()

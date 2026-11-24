import io
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

from scripts import ai_route_metrics


HEADER = "service,scenario,variant,connection,kind,outcome,connect_ms,tls_ready_ms,ttfb_ms,first_event_ms\n"


def summarize(*rows):
    return ai_route_metrics.measurements(io.StringIO(HEADER + "\n".join(rows)))


class AiRouteMetricsTests(unittest.TestCase):
    def test_stream_denominators_and_missing_timings(self):
        result, = summarize(
            "antigravity,ide-completion,A,unknown,stream,complete,,,80,110",
            "antigravity,ide-completion,A,unknown,stream,complete,,,,",
            "antigravity,ide-completion,A,unknown,stream,failed,,,,",
            "antigravity,ide-completion,A,unknown,stream,interrupted,,,90,120",
            "antigravity,ide-completion,A,unknown,stream,cancelled,,,100,125",
        )
        self.assertEqual(result["success"]["n"], 5)
        self.assertEqual(result["success"]["pct"], 40)
        self.assertEqual(result["failed"], 1)
        self.assertEqual(result["cancelled"], 1)
        self.assertEqual(result["stream_interruption"]["n"], 3)
        self.assertEqual(result["stream_interruption"]["pct"], 33.333)
        self.assertEqual(result["completed_latency_ms"]["ttfb_ms"], {"n": 1, "p95": 80})
        self.assertEqual(result["completed_latency_ms"]["first_event_ms"], {"n": 1, "p95": 110})
        self.assertEqual(result["completed_latency_ms"]["connect_ms"], {"n": 0, "p95": None})

    def test_nearest_rank_and_separate_groups(self):
        rows = [f"chatgpt,api,A,cold,request,complete,,,{i}," for i in range(1, 21)]
        groups = summarize(
            *rows,
            "chatgpt,api,B,cold,request,complete,,,100,",
            "chatgpt,api,A,warm,request,complete,,,5,",
            "gemini,api,B,cold,stream,failed,,,,",
        )
        self.assertEqual(len(groups), 4)
        self.assertEqual(groups[0]["completed_latency_ms"]["ttfb_ms"], {"n": 20, "p95": 19})
        self.assertNotIn("stream_interruption", groups[0])
        self.assertEqual(groups[0]["success"]["ci95_pct"], [83.887, 100.0])
        self.assertIsNone(groups[-1]["stream_interruption"]["pct"])

    def test_invalid_data_is_rejected(self):
        for row in (
            "claude,api,B,cold,request,complete,0,nan,,",
            "claude,api,B,cold,request,complete,0,,inf,",
            "claude,api,B,cold,request,complete,20,10,30,",
            "claude,api,B,cold,stream,failed,1,2,3,4",
            "claude,api,B,warm,request,interrupted,,,,",
            "claude,api,B,cold,stream,complete,0,0",
        ):
            with self.subTest(row=row), self.assertRaisesRegex(ValueError, "line 2"):
                summarize(row)
        with self.assertRaisesRegex(ValueError, "no request samples"):
            summarize()

    def test_cli_reads_bom_and_non_ascii_scenario(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "samples.csv"
            path.write_text(HEADER + "claude,caf\u00e9,B,cold,request,complete,10,20,30,", encoding="utf-8-sig")
            output = subprocess.check_output(
                [sys.executable, ai_route_metrics.__file__, str(path)], text=True, encoding="utf-8"
            )
        result, = json.loads(output)
        self.assertEqual(result["scenario"], "caf\u00e9")
        self.assertEqual(result["completed_latency_ms"]["tls_ready_ms"], {"n": 1, "p95": 20})

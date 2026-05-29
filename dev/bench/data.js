window.BENCHMARK_DATA = {
  "lastUpdate": 1780050132337,
  "repoUrl": "https://github.com/alex60217101990/qdf",
  "entries": {
    "qdf Go Benchmarks": [
      {
        "commit": {
          "author": {
            "email": "alex6021710@gmail.com",
            "name": "alex60217101990",
            "username": "alex60217101990"
          },
          "committer": {
            "email": "33520849+alex60217101990@users.noreply.github.com",
            "name": "alex60217101990",
            "username": "alex60217101990"
          },
          "distinct": true,
          "id": "1a22845310539658e6bb9195348945eea89a8c24",
          "message": "ci: set git identity for gh-pages bootstrap commit\n\nThe orphan-branch bootstrap ran git commit with no identity configured on\nthe runner, failing with 'empty ident name' (exit 128). Set an explicit\ngithub-actions[bot] identity on that commit.",
          "timestamp": "2026-05-29T13:13:47+03:00",
          "tree_id": "f78648f218445f3eb76efef2b7127ed0884de776",
          "url": "https://github.com/alex60217101990/qdf/commit/1a22845310539658e6bb9195348945eea89a8c24"
        },
        "date": 1780050132018,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json",
            "value": 163.2,
            "unit": "ns/op\t 153.18 MB/s\t      24 B/op\t       1 allocs/op",
            "extra": "14566993 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json - ns/op",
            "value": 163.2,
            "unit": "ns/op",
            "extra": "14566993 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json - MB/s",
            "value": 153.18,
            "unit": "MB/s",
            "extra": "14566993 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json - B/op",
            "value": 24,
            "unit": "B/op",
            "extra": "14566993 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "14566993 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal",
            "value": 190.2,
            "unit": "ns/op\t 126.20 MB/s\t      48 B/op\t       2 allocs/op",
            "extra": "12505351 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal - ns/op",
            "value": 190.2,
            "unit": "ns/op",
            "extra": "12505351 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal - MB/s",
            "value": 126.2,
            "unit": "MB/s",
            "extra": "12505351 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal - B/op",
            "value": 48,
            "unit": "B/op",
            "extra": "12505351 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "12505351 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack",
            "value": 241.2,
            "unit": "ns/op\t  66.35 MB/s\t     136 B/op\t       3 allocs/op",
            "extra": "9907884 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack - ns/op",
            "value": 241.2,
            "unit": "ns/op",
            "extra": "9907884 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack - MB/s",
            "value": 66.35,
            "unit": "MB/s",
            "extra": "9907884 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack - B/op",
            "value": 136,
            "unit": "B/op",
            "extra": "9907884 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "9907884 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast",
            "value": 309.2,
            "unit": "ns/op\t  71.16 MB/s\t      72 B/op\t       3 allocs/op",
            "extra": "7720291 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast - ns/op",
            "value": 309.2,
            "unit": "ns/op",
            "extra": "7720291 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast - MB/s",
            "value": 71.16,
            "unit": "MB/s",
            "extra": "7720291 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast - B/op",
            "value": 72,
            "unit": "B/op",
            "extra": "7720291 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "7720291 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense",
            "value": 389.4,
            "unit": "ns/op\t  64.20 MB/s\t      80 B/op\t       3 allocs/op",
            "extra": "6168367 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense - ns/op",
            "value": 389.4,
            "unit": "ns/op",
            "extra": "6168367 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense - MB/s",
            "value": 64.2,
            "unit": "MB/s",
            "extra": "6168367 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "6168367 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "6168367 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json",
            "value": 1006,
            "unit": "ns/op\t 209.67 MB/s\t     192 B/op\t       1 allocs/op",
            "extra": "2379606 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json - ns/op",
            "value": 1006,
            "unit": "ns/op",
            "extra": "2379606 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json - MB/s",
            "value": 209.67,
            "unit": "MB/s",
            "extra": "2379606 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json - B/op",
            "value": 192,
            "unit": "B/op",
            "extra": "2379606 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2379606 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal",
            "value": 1063,
            "unit": "ns/op\t 197.50 MB/s\t     416 B/op\t       2 allocs/op",
            "extra": "2226624 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal - ns/op",
            "value": 1063,
            "unit": "ns/op",
            "extra": "2226624 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal - MB/s",
            "value": 197.5,
            "unit": "MB/s",
            "extra": "2226624 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal - B/op",
            "value": 416,
            "unit": "B/op",
            "extra": "2226624 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "2226624 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack",
            "value": 1010,
            "unit": "ns/op\t 132.71 MB/s\t     688 B/op\t       5 allocs/op",
            "extra": "2381686 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack - ns/op",
            "value": 1010,
            "unit": "ns/op",
            "extra": "2381686 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack - MB/s",
            "value": 132.71,
            "unit": "MB/s",
            "extra": "2381686 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack - B/op",
            "value": 688,
            "unit": "B/op",
            "extra": "2381686 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "2381686 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast",
            "value": 595.3,
            "unit": "ns/op\t 221.74 MB/s\t     528 B/op\t       3 allocs/op",
            "extra": "4120819 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast - ns/op",
            "value": 595.3,
            "unit": "ns/op",
            "extra": "4120819 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast - MB/s",
            "value": 221.74,
            "unit": "MB/s",
            "extra": "4120819 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast - B/op",
            "value": 528,
            "unit": "B/op",
            "extra": "4120819 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "4120819 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense",
            "value": 759.6,
            "unit": "ns/op\t 181.68 MB/s\t     528 B/op\t       3 allocs/op",
            "extra": "3169310 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense - ns/op",
            "value": 759.6,
            "unit": "ns/op",
            "extra": "3169310 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense - MB/s",
            "value": 181.68,
            "unit": "MB/s",
            "extra": "3169310 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense - B/op",
            "value": 528,
            "unit": "B/op",
            "extra": "3169310 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3169310 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json",
            "value": 454.5,
            "unit": "ns/op\t 228.81 MB/s\t      80 B/op\t       1 allocs/op",
            "extra": "5266772 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json - ns/op",
            "value": 454.5,
            "unit": "ns/op",
            "extra": "5266772 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json - MB/s",
            "value": 228.81,
            "unit": "MB/s",
            "extra": "5266772 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "5266772 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "5266772 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal",
            "value": 481.3,
            "unit": "ns/op\t 214.01 MB/s\t     192 B/op\t       2 allocs/op",
            "extra": "4991608 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal - ns/op",
            "value": 481.3,
            "unit": "ns/op",
            "extra": "4991608 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal - MB/s",
            "value": 214.01,
            "unit": "MB/s",
            "extra": "4991608 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal - B/op",
            "value": 192,
            "unit": "B/op",
            "extra": "4991608 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "4991608 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack",
            "value": 722.7,
            "unit": "ns/op\t 105.15 MB/s\t     320 B/op\t       4 allocs/op",
            "extra": "3325437 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack - ns/op",
            "value": 722.7,
            "unit": "ns/op",
            "extra": "3325437 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack - MB/s",
            "value": 105.15,
            "unit": "MB/s",
            "extra": "3325437 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack - B/op",
            "value": 320,
            "unit": "B/op",
            "extra": "3325437 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "3325437 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast",
            "value": 441.5,
            "unit": "ns/op\t 194.77 MB/s\t     256 B/op\t       3 allocs/op",
            "extra": "5397732 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast - ns/op",
            "value": 441.5,
            "unit": "ns/op",
            "extra": "5397732 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast - MB/s",
            "value": 194.77,
            "unit": "MB/s",
            "extra": "5397732 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast - B/op",
            "value": 256,
            "unit": "B/op",
            "extra": "5397732 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "5397732 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense",
            "value": 653.8,
            "unit": "ns/op\t 146.84 MB/s\t     256 B/op\t       3 allocs/op",
            "extra": "3670419 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense - ns/op",
            "value": 653.8,
            "unit": "ns/op",
            "extra": "3670419 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense - MB/s",
            "value": 146.84,
            "unit": "MB/s",
            "extra": "3670419 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense - B/op",
            "value": 256,
            "unit": "B/op",
            "extra": "3670419 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3670419 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json",
            "value": 1463,
            "unit": "ns/op\t 164.02 MB/s\t       0 B/op\t       0 allocs/op",
            "extra": "1638819 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json - ns/op",
            "value": 1463,
            "unit": "ns/op",
            "extra": "1638819 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json - MB/s",
            "value": 164.02,
            "unit": "MB/s",
            "extra": "1638819 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1638819 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1638819 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal",
            "value": 1513,
            "unit": "ns/op\t 158.00 MB/s\t     240 B/op\t       1 allocs/op",
            "extra": "1584829 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal - ns/op",
            "value": 1513,
            "unit": "ns/op",
            "extra": "1584829 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal - MB/s",
            "value": 158,
            "unit": "MB/s",
            "extra": "1584829 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal - B/op",
            "value": 240,
            "unit": "B/op",
            "extra": "1584829 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "1584829 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack",
            "value": 2870,
            "unit": "ns/op\t  48.43 MB/s\t     752 B/op\t      20 allocs/op",
            "extra": "797553 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack - ns/op",
            "value": 2870,
            "unit": "ns/op",
            "extra": "797553 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack - MB/s",
            "value": 48.43,
            "unit": "MB/s",
            "extra": "797553 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack - B/op",
            "value": 752,
            "unit": "B/op",
            "extra": "797553 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "797553 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast",
            "value": 746.7,
            "unit": "ns/op\t 222.31 MB/s\t     176 B/op\t       1 allocs/op",
            "extra": "3213118 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast - ns/op",
            "value": 746.7,
            "unit": "ns/op",
            "extra": "3213118 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast - MB/s",
            "value": 222.31,
            "unit": "MB/s",
            "extra": "3213118 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast - B/op",
            "value": 176,
            "unit": "B/op",
            "extra": "3213118 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "3213118 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense",
            "value": 674.4,
            "unit": "ns/op\t  93.42 MB/s\t      64 B/op\t       1 allocs/op",
            "extra": "3541456 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense - ns/op",
            "value": 674.4,
            "unit": "ns/op",
            "extra": "3541456 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense - MB/s",
            "value": 93.42,
            "unit": "MB/s",
            "extra": "3541456 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense - B/op",
            "value": 64,
            "unit": "B/op",
            "extra": "3541456 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "3541456 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json",
            "value": 879106,
            "unit": "ns/op\t 242.18 MB/s\t     291 B/op\t       1 allocs/op",
            "extra": "2760 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json - ns/op",
            "value": 879106,
            "unit": "ns/op",
            "extra": "2760 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json - MB/s",
            "value": 242.18,
            "unit": "MB/s",
            "extra": "2760 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json - B/op",
            "value": 291,
            "unit": "B/op",
            "extra": "2760 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2760 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal",
            "value": 894478,
            "unit": "ns/op\t 238.02 MB/s\t  213440 B/op\t       2 allocs/op",
            "extra": "2676 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal - ns/op",
            "value": 894478,
            "unit": "ns/op",
            "extra": "2676 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal - MB/s",
            "value": 238.02,
            "unit": "MB/s",
            "extra": "2676 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal - B/op",
            "value": 213440,
            "unit": "B/op",
            "extra": "2676 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "2676 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack",
            "value": 767199,
            "unit": "ns/op\t 176.78 MB/s\t  524379 B/op\t      15 allocs/op",
            "extra": "3108 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack - ns/op",
            "value": 767199,
            "unit": "ns/op",
            "extra": "3108 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack - MB/s",
            "value": 176.78,
            "unit": "MB/s",
            "extra": "3108 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack - B/op",
            "value": 524379,
            "unit": "B/op",
            "extra": "3108 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "3108 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast",
            "value": 239056,
            "unit": "ns/op\t 538.08 MB/s\t  131157 B/op\t       3 allocs/op",
            "extra": "9650 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast - ns/op",
            "value": 239056,
            "unit": "ns/op",
            "extra": "9650 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast - MB/s",
            "value": 538.08,
            "unit": "MB/s",
            "extra": "9650 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast - B/op",
            "value": 131157,
            "unit": "B/op",
            "extra": "9650 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "9650 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense",
            "value": 156860,
            "unit": "ns/op\t 239.51 MB/s\t   42331 B/op\t      10 allocs/op",
            "extra": "15291 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense - ns/op",
            "value": 156860,
            "unit": "ns/op",
            "extra": "15291 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense - MB/s",
            "value": 239.51,
            "unit": "MB/s",
            "extra": "15291 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense - B/op",
            "value": 42331,
            "unit": "B/op",
            "extra": "15291 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "15291 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json",
            "value": 909523,
            "unit": "ns/op\t 271.46 MB/s\t   48345 B/op\t    1001 allocs/op",
            "extra": "2642 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json - ns/op",
            "value": 909523,
            "unit": "ns/op",
            "extra": "2642 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json - MB/s",
            "value": 271.46,
            "unit": "MB/s",
            "extra": "2642 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json - B/op",
            "value": 48345,
            "unit": "B/op",
            "extra": "2642 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json - allocs/op",
            "value": 1001,
            "unit": "allocs/op",
            "extra": "2642 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal",
            "value": 940857,
            "unit": "ns/op\t 262.42 MB/s\t  302907 B/op\t    1002 allocs/op",
            "extra": "2527 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal - ns/op",
            "value": 940857,
            "unit": "ns/op",
            "extra": "2527 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal - MB/s",
            "value": 262.42,
            "unit": "MB/s",
            "extra": "2527 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal - B/op",
            "value": 302907,
            "unit": "B/op",
            "extra": "2527 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal - allocs/op",
            "value": 1002,
            "unit": "allocs/op",
            "extra": "2527 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack",
            "value": 573293,
            "unit": "ns/op\t 323.81 MB/s\t  548384 B/op\t    1015 allocs/op",
            "extra": "4122 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack - ns/op",
            "value": 573293,
            "unit": "ns/op",
            "extra": "4122 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack - MB/s",
            "value": 323.81,
            "unit": "MB/s",
            "extra": "4122 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack - B/op",
            "value": 548384,
            "unit": "B/op",
            "extra": "4122 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack - allocs/op",
            "value": 1015,
            "unit": "allocs/op",
            "extra": "4122 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast",
            "value": 156466,
            "unit": "ns/op\t1186.51 MB/s\t  189085 B/op\t       3 allocs/op",
            "extra": "15268 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast - ns/op",
            "value": 156466,
            "unit": "ns/op",
            "extra": "15268 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast - MB/s",
            "value": 1186.51,
            "unit": "MB/s",
            "extra": "15268 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast - B/op",
            "value": 189085,
            "unit": "B/op",
            "extra": "15268 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "15268 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense",
            "value": 154840,
            "unit": "ns/op\t1198.98 MB/s\t  189250 B/op\t       3 allocs/op",
            "extra": "15506 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense - ns/op",
            "value": 154840,
            "unit": "ns/op",
            "extra": "15506 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense - MB/s",
            "value": 1198.98,
            "unit": "MB/s",
            "extra": "15506 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense - B/op",
            "value": 189250,
            "unit": "B/op",
            "extra": "15506 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "15506 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json",
            "value": 742.1,
            "unit": "ns/op\t  32.34 MB/s\t     248 B/op\t       6 allocs/op",
            "extra": "3226915 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json - ns/op",
            "value": 742.1,
            "unit": "ns/op",
            "extra": "3226915 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json - MB/s",
            "value": 32.34,
            "unit": "MB/s",
            "extra": "3226915 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json - B/op",
            "value": 248,
            "unit": "B/op",
            "extra": "3226915 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "3226915 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack",
            "value": 310,
            "unit": "ns/op\t  51.61 MB/s\t      77 B/op\t       3 allocs/op",
            "extra": "7738002 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack - ns/op",
            "value": 310,
            "unit": "ns/op",
            "extra": "7738002 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack - MB/s",
            "value": 51.61,
            "unit": "MB/s",
            "extra": "7738002 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack - B/op",
            "value": 77,
            "unit": "B/op",
            "extra": "7738002 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "7738002 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast",
            "value": 154.7,
            "unit": "ns/op\t 142.23 MB/s\t      29 B/op\t       2 allocs/op",
            "extra": "15395818 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast - ns/op",
            "value": 154.7,
            "unit": "ns/op",
            "extra": "15395818 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast - MB/s",
            "value": 142.23,
            "unit": "MB/s",
            "extra": "15395818 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast - B/op",
            "value": 29,
            "unit": "B/op",
            "extra": "15395818 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "15395818 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense",
            "value": 320.1,
            "unit": "ns/op\t  78.09 MB/s\t      72 B/op\t       4 allocs/op",
            "extra": "7453906 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense - ns/op",
            "value": 320.1,
            "unit": "ns/op",
            "extra": "7453906 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense - MB/s",
            "value": 78.09,
            "unit": "MB/s",
            "extra": "7453906 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense - B/op",
            "value": 72,
            "unit": "B/op",
            "extra": "7453906 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "7453906 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json",
            "value": 4566,
            "unit": "ns/op\t  45.99 MB/s\t     448 B/op\t      10 allocs/op",
            "extra": "507848 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json - ns/op",
            "value": 4566,
            "unit": "ns/op",
            "extra": "507848 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json - MB/s",
            "value": 45.99,
            "unit": "MB/s",
            "extra": "507848 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json - B/op",
            "value": 448,
            "unit": "B/op",
            "extra": "507848 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "507848 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack",
            "value": 1600,
            "unit": "ns/op\t  83.77 MB/s\t     272 B/op\t       7 allocs/op",
            "extra": "1503553 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack - ns/op",
            "value": 1600,
            "unit": "ns/op",
            "extra": "1503553 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack - MB/s",
            "value": 83.77,
            "unit": "MB/s",
            "extra": "1503553 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack - B/op",
            "value": 272,
            "unit": "B/op",
            "extra": "1503553 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "1503553 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast",
            "value": 814.9,
            "unit": "ns/op\t 161.98 MB/s\t     224 B/op\t       6 allocs/op",
            "extra": "2942827 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast - ns/op",
            "value": 814.9,
            "unit": "ns/op",
            "extra": "2942827 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast - MB/s",
            "value": 161.98,
            "unit": "MB/s",
            "extra": "2942827 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast - B/op",
            "value": 224,
            "unit": "B/op",
            "extra": "2942827 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "2942827 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense",
            "value": 1475,
            "unit": "ns/op\t  93.57 MB/s\t     624 B/op\t       8 allocs/op",
            "extra": "1629751 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense - ns/op",
            "value": 1475,
            "unit": "ns/op",
            "extra": "1629751 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense - MB/s",
            "value": 93.57,
            "unit": "MB/s",
            "extra": "1629751 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense - B/op",
            "value": 624,
            "unit": "B/op",
            "extra": "1629751 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "1629751 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json",
            "value": 2506,
            "unit": "ns/op\t  41.11 MB/s\t     664 B/op\t      15 allocs/op",
            "extra": "893797 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json - ns/op",
            "value": 2506,
            "unit": "ns/op",
            "extra": "893797 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json - MB/s",
            "value": 41.11,
            "unit": "MB/s",
            "extra": "893797 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json - B/op",
            "value": 664,
            "unit": "B/op",
            "extra": "893797 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "893797 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack",
            "value": 1088,
            "unit": "ns/op\t  69.88 MB/s\t     160 B/op\t       6 allocs/op",
            "extra": "2207010 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack - ns/op",
            "value": 1088,
            "unit": "ns/op",
            "extra": "2207010 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack - MB/s",
            "value": 69.88,
            "unit": "MB/s",
            "extra": "2207010 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack - B/op",
            "value": 160,
            "unit": "B/op",
            "extra": "2207010 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "2207010 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast",
            "value": 396.8,
            "unit": "ns/op\t 216.75 MB/s\t     112 B/op\t       5 allocs/op",
            "extra": "5984624 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast - ns/op",
            "value": 396.8,
            "unit": "ns/op",
            "extra": "5984624 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast - MB/s",
            "value": 216.75,
            "unit": "MB/s",
            "extra": "5984624 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast - B/op",
            "value": 112,
            "unit": "B/op",
            "extra": "5984624 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "5984624 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense",
            "value": 899.1,
            "unit": "ns/op\t 106.78 MB/s\t     296 B/op\t      15 allocs/op",
            "extra": "2688513 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense - ns/op",
            "value": 899.1,
            "unit": "ns/op",
            "extra": "2688513 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense - MB/s",
            "value": 106.78,
            "unit": "MB/s",
            "extra": "2688513 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense - B/op",
            "value": 296,
            "unit": "B/op",
            "extra": "2688513 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "2688513 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json",
            "value": 7031,
            "unit": "ns/op\t  33.99 MB/s\t    1200 B/op\t      29 allocs/op",
            "extra": "333363 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json - ns/op",
            "value": 7031,
            "unit": "ns/op",
            "extra": "333363 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json - MB/s",
            "value": 33.99,
            "unit": "MB/s",
            "extra": "333363 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json - B/op",
            "value": 1200,
            "unit": "B/op",
            "extra": "333363 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json - allocs/op",
            "value": 29,
            "unit": "allocs/op",
            "extra": "333363 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack",
            "value": 3861,
            "unit": "ns/op\t  36.00 MB/s\t     312 B/op\t      18 allocs/op",
            "extra": "611826 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack - ns/op",
            "value": 3861,
            "unit": "ns/op",
            "extra": "611826 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack - MB/s",
            "value": 36,
            "unit": "MB/s",
            "extra": "611826 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack - B/op",
            "value": 312,
            "unit": "B/op",
            "extra": "611826 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "611826 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast",
            "value": 1965,
            "unit": "ns/op\t  84.47 MB/s\t     264 B/op\t      17 allocs/op",
            "extra": "1220662 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast - ns/op",
            "value": 1965,
            "unit": "ns/op",
            "extra": "1220662 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast - MB/s",
            "value": 84.47,
            "unit": "MB/s",
            "extra": "1220662 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast - B/op",
            "value": 264,
            "unit": "B/op",
            "extra": "1220662 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "1220662 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense",
            "value": 1930,
            "unit": "ns/op\t  32.63 MB/s\t     304 B/op\t      19 allocs/op",
            "extra": "1242520 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense - ns/op",
            "value": 1930,
            "unit": "ns/op",
            "extra": "1242520 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense - MB/s",
            "value": 32.63,
            "unit": "MB/s",
            "extra": "1242520 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense - B/op",
            "value": 304,
            "unit": "B/op",
            "extra": "1242520 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense - allocs/op",
            "value": 19,
            "unit": "allocs/op",
            "extra": "1242520 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/json",
            "value": 4480115,
            "unit": "ns/op\t  47.52 MB/s\t  638351 B/op\t    5020 allocs/op",
            "extra": "534 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/json - ns/op",
            "value": 4480115,
            "unit": "ns/op",
            "extra": "534 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/json - MB/s",
            "value": 47.52,
            "unit": "MB/s",
            "extra": "534 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/json - B/op",
            "value": 638351,
            "unit": "B/op",
            "extra": "534 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/json - allocs/op",
            "value": 5020,
            "unit": "allocs/op",
            "extra": "534 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack",
            "value": 1543823,
            "unit": "ns/op\t  87.85 MB/s\t  409044 B/op\t    5007 allocs/op",
            "extra": "1518 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack - ns/op",
            "value": 1543823,
            "unit": "ns/op",
            "extra": "1518 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack - MB/s",
            "value": 87.85,
            "unit": "MB/s",
            "extra": "1518 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack - B/op",
            "value": 409044,
            "unit": "B/op",
            "extra": "1518 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack - allocs/op",
            "value": 5007,
            "unit": "allocs/op",
            "extra": "1518 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast",
            "value": 728855,
            "unit": "ns/op\t 176.49 MB/s\t  220500 B/op\t    5003 allocs/op",
            "extra": "3265 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast - ns/op",
            "value": 728855,
            "unit": "ns/op",
            "extra": "3265 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast - MB/s",
            "value": 176.49,
            "unit": "MB/s",
            "extra": "3265 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast - B/op",
            "value": 220500,
            "unit": "B/op",
            "extra": "3265 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast - allocs/op",
            "value": 5003,
            "unit": "allocs/op",
            "extra": "3265 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_dense",
            "value": 201223,
            "unit": "ns/op\t 186.71 MB/s\t  318265 B/op\t    5022 allocs/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_dense - ns/op",
            "value": 201223,
            "unit": "ns/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_dense - MB/s",
            "value": 186.71,
            "unit": "MB/s",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_dense - B/op",
            "value": 318265,
            "unit": "B/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_dense - allocs/op",
            "value": 5022,
            "unit": "allocs/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/json",
            "value": 3379274,
            "unit": "ns/op\t  73.06 MB/s\t  442536 B/op\t    7019 allocs/op",
            "extra": "704 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/json - ns/op",
            "value": 3379274,
            "unit": "ns/op",
            "extra": "704 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/json - MB/s",
            "value": 73.06,
            "unit": "MB/s",
            "extra": "704 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/json - B/op",
            "value": 442536,
            "unit": "B/op",
            "extra": "704 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/json - allocs/op",
            "value": 7019,
            "unit": "allocs/op",
            "extra": "704 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack",
            "value": 1069734,
            "unit": "ns/op\t 173.54 MB/s\t  407514 B/op\t    7007 allocs/op",
            "extra": "2190 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack - ns/op",
            "value": 1069734,
            "unit": "ns/op",
            "extra": "2190 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack - MB/s",
            "value": 173.54,
            "unit": "MB/s",
            "extra": "2190 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack - B/op",
            "value": 407514,
            "unit": "B/op",
            "extra": "2190 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack - allocs/op",
            "value": 7007,
            "unit": "allocs/op",
            "extra": "2190 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast",
            "value": 420015,
            "unit": "ns/op\t 442.01 MB/s\t  251713 B/op\t    7002 allocs/op",
            "extra": "5654 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast - ns/op",
            "value": 420015,
            "unit": "ns/op",
            "extra": "5654 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast - MB/s",
            "value": 442.01,
            "unit": "MB/s",
            "extra": "5654 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast - B/op",
            "value": 251713,
            "unit": "B/op",
            "extra": "5654 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast - allocs/op",
            "value": 7002,
            "unit": "allocs/op",
            "extra": "5654 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense",
            "value": 419963,
            "unit": "ns/op\t 442.06 MB/s\t  255168 B/op\t    7005 allocs/op",
            "extra": "5634 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense - ns/op",
            "value": 419963,
            "unit": "ns/op",
            "extra": "5634 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense - MB/s",
            "value": 442.06,
            "unit": "MB/s",
            "extra": "5634 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense - B/op",
            "value": 255168,
            "unit": "B/op",
            "extra": "5634 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense - allocs/op",
            "value": 7005,
            "unit": "allocs/op",
            "extra": "5634 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen",
            "value": 314163,
            "unit": "ns/op\t 590.93 MB/s\t  908028 B/op\t      26 allocs/op",
            "extra": "7820 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen - ns/op",
            "value": 314163,
            "unit": "ns/op",
            "extra": "7820 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen - MB/s",
            "value": 590.93,
            "unit": "MB/s",
            "extra": "7820 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen - B/op",
            "value": 908028,
            "unit": "B/op",
            "extra": "7820 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen - allocs/op",
            "value": 26,
            "unit": "allocs/op",
            "extra": "7820 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen",
            "value": 417532,
            "unit": "ns/op\t 444.63 MB/s\t  251648 B/op\t    7001 allocs/op",
            "extra": "5491 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen - ns/op",
            "value": 417532,
            "unit": "ns/op",
            "extra": "5491 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen - MB/s",
            "value": 444.63,
            "unit": "MB/s",
            "extra": "5491 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen - B/op",
            "value": 251648,
            "unit": "B/op",
            "extra": "5491 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen - allocs/op",
            "value": 7001,
            "unit": "allocs/op",
            "extra": "5491 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json",
            "value": 140822,
            "unit": "ns/op\t 191.20 MB/s\t   27435 B/op\t       2 allocs/op",
            "extra": "17038 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json - ns/op",
            "value": 140822,
            "unit": "ns/op",
            "extra": "17038 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json - MB/s",
            "value": 191.2,
            "unit": "MB/s",
            "extra": "17038 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json - B/op",
            "value": 27435,
            "unit": "B/op",
            "extra": "17038 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "17038 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack",
            "value": 170503,
            "unit": "ns/op\t 222.83 MB/s\t  131235 B/op\t      13 allocs/op",
            "extra": "14067 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack - ns/op",
            "value": 170503,
            "unit": "ns/op",
            "extra": "14067 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack - MB/s",
            "value": 222.83,
            "unit": "MB/s",
            "extra": "14067 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack - B/op",
            "value": 131235,
            "unit": "B/op",
            "extra": "14067 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack - allocs/op",
            "value": 13,
            "unit": "allocs/op",
            "extra": "14067 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf",
            "value": 14776,
            "unit": "ns/op\t 592.78 MB/s\t    9795 B/op\t       3 allocs/op",
            "extra": "161276 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf - ns/op",
            "value": 14776,
            "unit": "ns/op",
            "extra": "161276 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf - MB/s",
            "value": 592.78,
            "unit": "MB/s",
            "extra": "161276 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf - B/op",
            "value": 9795,
            "unit": "B/op",
            "extra": "161276 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "161276 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json",
            "value": 626469,
            "unit": "ns/op\t  42.98 MB/s\t  104576 B/op\t      65 allocs/op",
            "extra": "3772 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json - ns/op",
            "value": 626469,
            "unit": "ns/op",
            "extra": "3772 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json - MB/s",
            "value": 42.98,
            "unit": "MB/s",
            "extra": "3772 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json - B/op",
            "value": 104576,
            "unit": "B/op",
            "extra": "3772 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json - allocs/op",
            "value": 65,
            "unit": "allocs/op",
            "extra": "3772 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack",
            "value": 242697,
            "unit": "ns/op\t 156.55 MB/s\t   68194 B/op\t      29 allocs/op",
            "extra": "9427 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack - ns/op",
            "value": 242697,
            "unit": "ns/op",
            "extra": "9427 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack - MB/s",
            "value": 156.55,
            "unit": "MB/s",
            "extra": "9427 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack - B/op",
            "value": 68194,
            "unit": "B/op",
            "extra": "9427 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack - allocs/op",
            "value": 29,
            "unit": "allocs/op",
            "extra": "9427 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf",
            "value": 12095,
            "unit": "ns/op\t 724.18 MB/s\t   42332 B/op\t      11 allocs/op",
            "extra": "198084 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf - ns/op",
            "value": 12095,
            "unit": "ns/op",
            "extra": "198084 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf - MB/s",
            "value": 724.18,
            "unit": "MB/s",
            "extra": "198084 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf - B/op",
            "value": 42332,
            "unit": "B/op",
            "extra": "198084 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "198084 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json",
            "value": 70108,
            "unit": "ns/op\t 246.96 MB/s\t   18532 B/op\t       2 allocs/op",
            "extra": "34152 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json - ns/op",
            "value": 70108,
            "unit": "ns/op",
            "extra": "34152 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json - MB/s",
            "value": 246.96,
            "unit": "MB/s",
            "extra": "34152 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json - B/op",
            "value": 18532,
            "unit": "B/op",
            "extra": "34152 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "34152 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack",
            "value": 104125,
            "unit": "ns/op\t 266.12 MB/s\t   65625 B/op\t      12 allocs/op",
            "extra": "23002 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack - ns/op",
            "value": 104125,
            "unit": "ns/op",
            "extra": "23002 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack - MB/s",
            "value": 266.12,
            "unit": "MB/s",
            "extra": "23002 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack - B/op",
            "value": 65625,
            "unit": "B/op",
            "extra": "23002 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "23002 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf",
            "value": 12186,
            "unit": "ns/op\t  46.20 MB/s\t     768 B/op\t       3 allocs/op",
            "extra": "195158 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf - ns/op",
            "value": 12186,
            "unit": "ns/op",
            "extra": "195158 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf - MB/s",
            "value": 46.2,
            "unit": "MB/s",
            "extra": "195158 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf - B/op",
            "value": 768,
            "unit": "B/op",
            "extra": "195158 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "195158 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json",
            "value": 387009,
            "unit": "ns/op\t  44.74 MB/s\t   75976 B/op\t      43 allocs/op",
            "extra": "5959 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json - ns/op",
            "value": 387009,
            "unit": "ns/op",
            "extra": "5959 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json - MB/s",
            "value": 44.74,
            "unit": "MB/s",
            "extra": "5959 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json - B/op",
            "value": 75976,
            "unit": "B/op",
            "extra": "5959 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json - allocs/op",
            "value": 43,
            "unit": "allocs/op",
            "extra": "5959 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack",
            "value": 159750,
            "unit": "ns/op\t 173.46 MB/s\t   49543 B/op\t      18 allocs/op",
            "extra": "14992 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack - ns/op",
            "value": 159750,
            "unit": "ns/op",
            "extra": "14992 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack - MB/s",
            "value": 173.46,
            "unit": "MB/s",
            "extra": "14992 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack - B/op",
            "value": 49543,
            "unit": "B/op",
            "extra": "14992 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "14992 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf",
            "value": 9556,
            "unit": "ns/op\t  58.91 MB/s\t   32895 B/op\t       6 allocs/op",
            "extra": "254282 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf - ns/op",
            "value": 9556,
            "unit": "ns/op",
            "extra": "254282 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf - MB/s",
            "value": 58.91,
            "unit": "MB/s",
            "extra": "254282 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf - B/op",
            "value": 32895,
            "unit": "B/op",
            "extra": "254282 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "254282 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json",
            "value": 27581,
            "unit": "ns/op\t 247.53 MB/s\t    6962 B/op\t       2 allocs/op",
            "extra": "86584 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json - ns/op",
            "value": 27581,
            "unit": "ns/op",
            "extra": "86584 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json - MB/s",
            "value": 247.53,
            "unit": "MB/s",
            "extra": "86584 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json - B/op",
            "value": 6962,
            "unit": "B/op",
            "extra": "86584 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "86584 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack",
            "value": 38636,
            "unit": "ns/op\t 239.39 MB/s\t   32804 B/op\t      11 allocs/op",
            "extra": "62286 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack - ns/op",
            "value": 38636,
            "unit": "ns/op",
            "extra": "62286 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack - MB/s",
            "value": 239.39,
            "unit": "MB/s",
            "extra": "62286 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack - B/op",
            "value": 32804,
            "unit": "B/op",
            "extra": "62286 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "62286 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf",
            "value": 11750,
            "unit": "ns/op\t  26.21 MB/s\t     416 B/op\t       3 allocs/op",
            "extra": "202858 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf - ns/op",
            "value": 11750,
            "unit": "ns/op",
            "extra": "202858 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf - MB/s",
            "value": 26.21,
            "unit": "MB/s",
            "extra": "202858 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf - B/op",
            "value": 416,
            "unit": "B/op",
            "extra": "202858 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "202858 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json",
            "value": 148371,
            "unit": "ns/op\t  46.01 MB/s\t   25504 B/op\t      19 allocs/op",
            "extra": "16178 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json - ns/op",
            "value": 148371,
            "unit": "ns/op",
            "extra": "16178 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json - MB/s",
            "value": 46.01,
            "unit": "MB/s",
            "extra": "16178 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json - B/op",
            "value": 25504,
            "unit": "B/op",
            "extra": "16178 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json - allocs/op",
            "value": 19,
            "unit": "allocs/op",
            "extra": "16178 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack",
            "value": 53562,
            "unit": "ns/op\t 172.68 MB/s\t   16570 B/op\t       8 allocs/op",
            "extra": "44799 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack - ns/op",
            "value": 53562,
            "unit": "ns/op",
            "extra": "44799 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack - MB/s",
            "value": 172.68,
            "unit": "MB/s",
            "extra": "44799 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack - B/op",
            "value": 16570,
            "unit": "B/op",
            "extra": "44799 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "44799 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf",
            "value": 5556,
            "unit": "ns/op\t  55.44 MB/s\t   16451 B/op\t       4 allocs/op",
            "extra": "424382 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf - ns/op",
            "value": 5556,
            "unit": "ns/op",
            "extra": "424382 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf - MB/s",
            "value": 55.44,
            "unit": "MB/s",
            "extra": "424382 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf - B/op",
            "value": 16451,
            "unit": "B/op",
            "extra": "424382 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "424382 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json",
            "value": 169669,
            "unit": "ns/op\t 432.93 MB/s\t   73837 B/op\t       2 allocs/op",
            "extra": "14131 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json - ns/op",
            "value": 169669,
            "unit": "ns/op",
            "extra": "14131 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json - MB/s",
            "value": 432.93,
            "unit": "MB/s",
            "extra": "14131 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json - B/op",
            "value": 73837,
            "unit": "B/op",
            "extra": "14131 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "14131 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack",
            "value": 145137,
            "unit": "ns/op\t 411.42 MB/s\t  131101 B/op\t      13 allocs/op",
            "extra": "16611 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack - ns/op",
            "value": 145137,
            "unit": "ns/op",
            "extra": "16611 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack - MB/s",
            "value": 411.42,
            "unit": "MB/s",
            "extra": "16611 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack - B/op",
            "value": 131101,
            "unit": "B/op",
            "extra": "16611 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack - allocs/op",
            "value": 13,
            "unit": "allocs/op",
            "extra": "16611 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf",
            "value": 89487,
            "unit": "ns/op\t 375.55 MB/s\t   41117 B/op\t       3 allocs/op",
            "extra": "26007 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf - ns/op",
            "value": 89487,
            "unit": "ns/op",
            "extra": "26007 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf - MB/s",
            "value": 375.55,
            "unit": "MB/s",
            "extra": "26007 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf - B/op",
            "value": 41117,
            "unit": "B/op",
            "extra": "26007 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "26007 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json",
            "value": 976225,
            "unit": "ns/op\t  75.24 MB/s\t  125256 B/op\t    2016 allocs/op",
            "extra": "2424 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json - ns/op",
            "value": 976225,
            "unit": "ns/op",
            "extra": "2424 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json - MB/s",
            "value": 75.24,
            "unit": "MB/s",
            "extra": "2424 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json - B/op",
            "value": 125256,
            "unit": "B/op",
            "extra": "2424 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json - allocs/op",
            "value": 2016,
            "unit": "allocs/op",
            "extra": "2424 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack",
            "value": 288686,
            "unit": "ns/op\t 206.84 MB/s\t  114785 B/op\t    2007 allocs/op",
            "extra": "8161 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack - ns/op",
            "value": 288686,
            "unit": "ns/op",
            "extra": "8161 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack - MB/s",
            "value": 206.84,
            "unit": "MB/s",
            "extra": "8161 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack - B/op",
            "value": 114785,
            "unit": "B/op",
            "extra": "8161 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack - allocs/op",
            "value": 2007,
            "unit": "allocs/op",
            "extra": "8161 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf",
            "value": 99015,
            "unit": "ns/op\t 339.41 MB/s\t   65197 B/op\t    1012 allocs/op",
            "extra": "24114 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf - ns/op",
            "value": 99015,
            "unit": "ns/op",
            "extra": "24114 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf - MB/s",
            "value": 339.41,
            "unit": "MB/s",
            "extra": "24114 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf - B/op",
            "value": 65197,
            "unit": "B/op",
            "extra": "24114 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf - allocs/op",
            "value": 1012,
            "unit": "allocs/op",
            "extra": "24114 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/json",
            "value": 26276,
            "unit": "ns/op\t    2736 B/op\t       2 allocs/op",
            "extra": "90130 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/json - ns/op",
            "value": 26276,
            "unit": "ns/op",
            "extra": "90130 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/json - B/op",
            "value": 2736,
            "unit": "B/op",
            "extra": "90130 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "90130 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/msgpack",
            "value": 18103,
            "unit": "ns/op\t    8225 B/op\t       9 allocs/op",
            "extra": "131642 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/msgpack - ns/op",
            "value": 18103,
            "unit": "ns/op",
            "extra": "131642 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/msgpack - B/op",
            "value": 8225,
            "unit": "B/op",
            "extra": "131642 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/msgpack - allocs/op",
            "value": 9,
            "unit": "allocs/op",
            "extra": "131642 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/qdf_fast",
            "value": 1918,
            "unit": "ns/op\t    2784 B/op\t       3 allocs/op",
            "extra": "1250966 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/qdf_fast - ns/op",
            "value": 1918,
            "unit": "ns/op",
            "extra": "1250966 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/qdf_fast - B/op",
            "value": 2784,
            "unit": "B/op",
            "extra": "1250966 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1250966 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/json",
            "value": 29426,
            "unit": "ns/op\t    2736 B/op\t       2 allocs/op",
            "extra": "81813 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/json - ns/op",
            "value": 29426,
            "unit": "ns/op",
            "extra": "81813 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/json - B/op",
            "value": 2736,
            "unit": "B/op",
            "extra": "81813 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "81813 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/msgpack",
            "value": 19861,
            "unit": "ns/op\t   16418 B/op\t      10 allocs/op",
            "extra": "122137 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/msgpack - ns/op",
            "value": 19861,
            "unit": "ns/op",
            "extra": "122137 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/msgpack - B/op",
            "value": 16418,
            "unit": "B/op",
            "extra": "122137 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/msgpack - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "122137 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/qdf_fast",
            "value": 2220,
            "unit": "ns/op\t    4961 B/op\t       3 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/qdf_fast - ns/op",
            "value": 2220,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/qdf_fast - B/op",
            "value": 4961,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/json",
            "value": 75011,
            "unit": "ns/op\t    4384 B/op\t      16 allocs/op",
            "extra": "32000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/json - ns/op",
            "value": 75011,
            "unit": "ns/op",
            "extra": "32000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/json - B/op",
            "value": 4384,
            "unit": "B/op",
            "extra": "32000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/json - allocs/op",
            "value": 16,
            "unit": "allocs/op",
            "extra": "32000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/msgpack",
            "value": 25708,
            "unit": "ns/op\t    4280 B/op\t       8 allocs/op",
            "extra": "92400 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/msgpack - ns/op",
            "value": 25708,
            "unit": "ns/op",
            "extra": "92400 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/msgpack - B/op",
            "value": 4280,
            "unit": "B/op",
            "extra": "92400 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/msgpack - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "92400 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/qdf_fast",
            "value": 3760,
            "unit": "ns/op\t    2112 B/op\t       3 allocs/op",
            "extra": "648339 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/qdf_fast - ns/op",
            "value": 3760,
            "unit": "ns/op",
            "extra": "648339 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/qdf_fast - B/op",
            "value": 2112,
            "unit": "B/op",
            "extra": "648339 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "648339 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/json",
            "value": 153002525,
            "unit": "ns/op\t 243.38 MB/s\t57769314 B/op\t  350217 allocs/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/json - ns/op",
            "value": 153002525,
            "unit": "ns/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/json - MB/s",
            "value": 243.38,
            "unit": "MB/s",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/json - B/op",
            "value": 57769314,
            "unit": "B/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/json - allocs/op",
            "value": 350217,
            "unit": "allocs/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/msgpack",
            "value": 84130512,
            "unit": "ns/op\t 289.66 MB/s\t68709102 B/op\t  100022 allocs/op",
            "extra": "28 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/msgpack - ns/op",
            "value": 84130512,
            "unit": "ns/op",
            "extra": "28 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/msgpack - MB/s",
            "value": 289.66,
            "unit": "MB/s",
            "extra": "28 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/msgpack - B/op",
            "value": 68709102,
            "unit": "B/op",
            "extra": "28 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/msgpack - allocs/op",
            "value": 100022,
            "unit": "allocs/op",
            "extra": "28 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_fast",
            "value": 27154559,
            "unit": "ns/op\t 887.79 MB/s\t29512042 B/op\t      19 allocs/op",
            "extra": "90 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_fast - ns/op",
            "value": 27154559,
            "unit": "ns/op",
            "extra": "90 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_fast - MB/s",
            "value": 887.79,
            "unit": "MB/s",
            "extra": "90 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_fast - B/op",
            "value": 29512042,
            "unit": "B/op",
            "extra": "90 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_fast - allocs/op",
            "value": 19,
            "unit": "allocs/op",
            "extra": "90 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_qpack",
            "value": 24249734,
            "unit": "ns/op\t 968.89 MB/s\t28815726 B/op\t      19 allocs/op",
            "extra": "100 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_qpack - ns/op",
            "value": 24249734,
            "unit": "ns/op",
            "extra": "100 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_qpack - MB/s",
            "value": 968.89,
            "unit": "MB/s",
            "extra": "100 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_qpack - B/op",
            "value": 28815726,
            "unit": "B/op",
            "extra": "100 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_qpack - allocs/op",
            "value": 19,
            "unit": "allocs/op",
            "extra": "100 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_dense",
            "value": 26691337,
            "unit": "ns/op\t 677.59 MB/s\t24114320 B/op\t      74 allocs/op",
            "extra": "88 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_dense - ns/op",
            "value": 26691337,
            "unit": "ns/op",
            "extra": "88 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_dense - MB/s",
            "value": 677.59,
            "unit": "MB/s",
            "extra": "88 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_dense - B/op",
            "value": 24114320,
            "unit": "B/op",
            "extra": "88 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_dense - allocs/op",
            "value": 74,
            "unit": "allocs/op",
            "extra": "88 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/json",
            "value": 614075153,
            "unit": "ns/op\t  60.65 MB/s\t119804008 B/op\t 1559637 allocs/op",
            "extra": "4 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/json - ns/op",
            "value": 614075153,
            "unit": "ns/op",
            "extra": "4 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/json - MB/s",
            "value": 60.65,
            "unit": "MB/s",
            "extra": "4 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/json - B/op",
            "value": 119804008,
            "unit": "B/op",
            "extra": "4 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/json - allocs/op",
            "value": 1559637,
            "unit": "allocs/op",
            "extra": "4 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/msgpack",
            "value": 184285015,
            "unit": "ns/op\t 132.26 MB/s\t74390932 B/op\t 1425125 allocs/op",
            "extra": "12 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/msgpack - ns/op",
            "value": 184285015,
            "unit": "ns/op",
            "extra": "12 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/msgpack - MB/s",
            "value": 132.26,
            "unit": "MB/s",
            "extra": "12 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/msgpack - B/op",
            "value": 74390932,
            "unit": "B/op",
            "extra": "12 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/msgpack - allocs/op",
            "value": 1425125,
            "unit": "allocs/op",
            "extra": "12 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_fast",
            "value": 73133700,
            "unit": "ns/op\t 329.69 MB/s\t48380968 B/op\t  875099 allocs/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_fast - ns/op",
            "value": 73133700,
            "unit": "ns/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_fast - MB/s",
            "value": 329.69,
            "unit": "MB/s",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_fast - B/op",
            "value": 48380968,
            "unit": "B/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_fast - allocs/op",
            "value": 875099,
            "unit": "allocs/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_qpack",
            "value": 67410780,
            "unit": "ns/op\t 348.57 MB/s\t48380218 B/op\t  875099 allocs/op",
            "extra": "33 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_qpack - ns/op",
            "value": 67410780,
            "unit": "ns/op",
            "extra": "33 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_qpack - MB/s",
            "value": 348.57,
            "unit": "MB/s",
            "extra": "33 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_qpack - B/op",
            "value": 48380218,
            "unit": "B/op",
            "extra": "33 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_qpack - allocs/op",
            "value": 875099,
            "unit": "allocs/op",
            "extra": "33 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense",
            "value": 61200664,
            "unit": "ns/op\t 295.55 MB/s\t50892365 B/op\t  790950 allocs/op",
            "extra": "39 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense - ns/op",
            "value": 61200664,
            "unit": "ns/op",
            "extra": "39 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense - MB/s",
            "value": 295.55,
            "unit": "MB/s",
            "extra": "39 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense - B/op",
            "value": 50892365,
            "unit": "B/op",
            "extra": "39 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense - allocs/op",
            "value": 790950,
            "unit": "allocs/op",
            "extra": "39 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/json",
            "value": 7938,
            "unit": "ns/op\t    3408 B/op\t      84 allocs/op",
            "extra": "298755 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/json - ns/op",
            "value": 7938,
            "unit": "ns/op",
            "extra": "298755 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/json - B/op",
            "value": 3408,
            "unit": "B/op",
            "extra": "298755 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/json - allocs/op",
            "value": 84,
            "unit": "allocs/op",
            "extra": "298755 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/msgpack",
            "value": 4397,
            "unit": "ns/op\t    1536 B/op\t      46 allocs/op",
            "extra": "531028 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/msgpack - ns/op",
            "value": 4397,
            "unit": "ns/op",
            "extra": "531028 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/msgpack - B/op",
            "value": 1536,
            "unit": "B/op",
            "extra": "531028 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/msgpack - allocs/op",
            "value": 46,
            "unit": "allocs/op",
            "extra": "531028 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_fast",
            "value": 1474,
            "unit": "ns/op\t     416 B/op\t       3 allocs/op",
            "extra": "1627510 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_fast - ns/op",
            "value": 1474,
            "unit": "ns/op",
            "extra": "1627510 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_fast - B/op",
            "value": 416,
            "unit": "B/op",
            "extra": "1627510 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1627510 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_dense",
            "value": 1723,
            "unit": "ns/op\t     416 B/op\t       3 allocs/op",
            "extra": "1392165 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_dense - ns/op",
            "value": 1723,
            "unit": "ns/op",
            "extra": "1392165 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_dense - B/op",
            "value": 416,
            "unit": "B/op",
            "extra": "1392165 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1392165 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/json",
            "value": 17005,
            "unit": "ns/op\t    4912 B/op\t     124 allocs/op",
            "extra": "140252 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/json - ns/op",
            "value": 17005,
            "unit": "ns/op",
            "extra": "140252 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/json - B/op",
            "value": 4912,
            "unit": "B/op",
            "extra": "140252 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/json - allocs/op",
            "value": 124,
            "unit": "allocs/op",
            "extra": "140252 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/msgpack",
            "value": 7257,
            "unit": "ns/op\t    3088 B/op\t     112 allocs/op",
            "extra": "331738 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/msgpack - ns/op",
            "value": 7257,
            "unit": "ns/op",
            "extra": "331738 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/msgpack - B/op",
            "value": 3088,
            "unit": "B/op",
            "extra": "331738 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/msgpack - allocs/op",
            "value": 112,
            "unit": "allocs/op",
            "extra": "331738 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/qdf_fast",
            "value": 2909,
            "unit": "ns/op\t    2356 B/op\t      32 allocs/op",
            "extra": "795092 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/qdf_fast - ns/op",
            "value": 2909,
            "unit": "ns/op",
            "extra": "795092 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/qdf_fast - B/op",
            "value": 2356,
            "unit": "B/op",
            "extra": "795092 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/qdf_fast - allocs/op",
            "value": 32,
            "unit": "allocs/op",
            "extra": "795092 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/json",
            "value": 9225,
            "unit": "ns/op\t    2820 B/op\t      71 allocs/op",
            "extra": "262873 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/json - ns/op",
            "value": 9225,
            "unit": "ns/op",
            "extra": "262873 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/json - B/op",
            "value": 2820,
            "unit": "B/op",
            "extra": "262873 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/json - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "262873 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/msgpack",
            "value": 2364,
            "unit": "ns/op\t    1487 B/op\t      46 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/msgpack - ns/op",
            "value": 2364,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/msgpack - B/op",
            "value": 1487,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/msgpack - allocs/op",
            "value": 46,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/qdf_fast",
            "value": 1648,
            "unit": "ns/op\t    1403 B/op\t      25 allocs/op",
            "extra": "1453838 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/qdf_fast - ns/op",
            "value": 1648,
            "unit": "ns/op",
            "extra": "1453838 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/qdf_fast - B/op",
            "value": 1403,
            "unit": "B/op",
            "extra": "1453838 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/qdf_fast - allocs/op",
            "value": 25,
            "unit": "allocs/op",
            "extra": "1453838 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/json",
            "value": 0.3513,
            "unit": "ns/op\t    442536 B/decode\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/json - ns/op",
            "value": 0.3513,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/json - B/decode",
            "value": 442536,
            "unit": "B/decode",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/json - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/json - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/msgpack",
            "value": 0.113,
            "unit": "ns/op\t    407515 B/decode\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/msgpack - ns/op",
            "value": 0.113,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/msgpack - B/decode",
            "value": 407515,
            "unit": "B/decode",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/msgpack - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/msgpack - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/qdf_fast",
            "value": 0.04386,
            "unit": "ns/op\t    251763 B/decode\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/qdf_fast - ns/op",
            "value": 0.04386,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/qdf_fast - B/decode",
            "value": 251763,
            "unit": "B/decode",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/qdf_fast - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/qdf_fast - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/json",
            "value": 3457,
            "unit": "ns/op\t     790 B/op\t      37 allocs/op",
            "extra": "666507 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/json - ns/op",
            "value": 3457,
            "unit": "ns/op",
            "extra": "666507 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/json - B/op",
            "value": 790,
            "unit": "B/op",
            "extra": "666507 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/json - allocs/op",
            "value": 37,
            "unit": "allocs/op",
            "extra": "666507 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/qdf_fast",
            "value": 604.1,
            "unit": "ns/op\t     347 B/op\t       3 allocs/op",
            "extra": "3906375 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/qdf_fast - ns/op",
            "value": 604.1,
            "unit": "ns/op",
            "extra": "3906375 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/qdf_fast - B/op",
            "value": 347,
            "unit": "B/op",
            "extra": "3906375 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3906375 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json",
            "value": 624.5,
            "unit": "ns/op\t 155.32 MB/s\t     240 B/op\t       3 allocs/op",
            "extra": "3849607 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json - ns/op",
            "value": 624.5,
            "unit": "ns/op",
            "extra": "3849607 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json - MB/s",
            "value": 155.32,
            "unit": "MB/s",
            "extra": "3849607 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json - B/op",
            "value": 240,
            "unit": "B/op",
            "extra": "3849607 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3849607 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack",
            "value": 388.2,
            "unit": "ns/op\t 162.29 MB/s\t     192 B/op\t       3 allocs/op",
            "extra": "6169060 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack - ns/op",
            "value": 388.2,
            "unit": "ns/op",
            "extra": "6169060 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack - MB/s",
            "value": 162.29,
            "unit": "MB/s",
            "extra": "6169060 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack - B/op",
            "value": 192,
            "unit": "B/op",
            "extra": "6169060 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "6169060 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf",
            "value": 391.6,
            "unit": "ns/op\t 183.86 MB/s\t     240 B/op\t       3 allocs/op",
            "extra": "6126162 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf - ns/op",
            "value": 391.6,
            "unit": "ns/op",
            "extra": "6126162 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf - MB/s",
            "value": 183.86,
            "unit": "MB/s",
            "extra": "6126162 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf - B/op",
            "value": 240,
            "unit": "B/op",
            "extra": "6126162 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "6126162 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json",
            "value": 1680,
            "unit": "ns/op\t  57.73 MB/s\t     328 B/op\t       7 allocs/op",
            "extra": "1426971 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json - ns/op",
            "value": 1680,
            "unit": "ns/op",
            "extra": "1426971 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json - MB/s",
            "value": 57.73,
            "unit": "MB/s",
            "extra": "1426971 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json - B/op",
            "value": 328,
            "unit": "B/op",
            "extra": "1426971 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "1426971 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack",
            "value": 617,
            "unit": "ns/op\t 102.10 MB/s\t     160 B/op\t       4 allocs/op",
            "extra": "3900285 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack - ns/op",
            "value": 617,
            "unit": "ns/op",
            "extra": "3900285 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack - MB/s",
            "value": 102.1,
            "unit": "MB/s",
            "extra": "3900285 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack - B/op",
            "value": 160,
            "unit": "B/op",
            "extra": "3900285 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "3900285 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf",
            "value": 274.4,
            "unit": "ns/op\t 262.41 MB/s\t     112 B/op\t       3 allocs/op",
            "extra": "8759960 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf - ns/op",
            "value": 274.4,
            "unit": "ns/op",
            "extra": "8759960 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf - MB/s",
            "value": 262.41,
            "unit": "MB/s",
            "extra": "8759960 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf - B/op",
            "value": 112,
            "unit": "B/op",
            "extra": "8759960 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "8759960 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json",
            "value": 366791,
            "unit": "ns/op\t 389.54 MB/s\t  149469 B/op\t       2 allocs/op",
            "extra": "6414 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json - ns/op",
            "value": 366791,
            "unit": "ns/op",
            "extra": "6414 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json - MB/s",
            "value": 389.54,
            "unit": "MB/s",
            "extra": "6414 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json - B/op",
            "value": 149469,
            "unit": "B/op",
            "extra": "6414 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "6414 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack",
            "value": 463557,
            "unit": "ns/op\t 240.83 MB/s\t  286202 B/op\t    1014 allocs/op",
            "extra": "5073 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack - ns/op",
            "value": 463557,
            "unit": "ns/op",
            "extra": "5073 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack - MB/s",
            "value": 240.83,
            "unit": "MB/s",
            "extra": "5073 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack - B/op",
            "value": 286202,
            "unit": "B/op",
            "extra": "5073 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack - allocs/op",
            "value": 1014,
            "unit": "allocs/op",
            "extra": "5073 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf",
            "value": 255535,
            "unit": "ns/op\t 158.74 MB/s\t   41106 B/op\t       3 allocs/op",
            "extra": "9134 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf - ns/op",
            "value": 255535,
            "unit": "ns/op",
            "extra": "9134 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf - MB/s",
            "value": 158.74,
            "unit": "MB/s",
            "extra": "9134 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf - B/op",
            "value": 41106,
            "unit": "B/op",
            "extra": "9134 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "9134 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json",
            "value": 2640879,
            "unit": "ns/op\t  54.10 MB/s\t  503579 B/op\t    9019 allocs/op",
            "extra": "907 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json - ns/op",
            "value": 2640879,
            "unit": "ns/op",
            "extra": "907 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json - MB/s",
            "value": 54.1,
            "unit": "MB/s",
            "extra": "907 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json - B/op",
            "value": 503579,
            "unit": "B/op",
            "extra": "907 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json - allocs/op",
            "value": 9019,
            "unit": "allocs/op",
            "extra": "907 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack",
            "value": 936279,
            "unit": "ns/op\t 119.23 MB/s\t  323891 B/op\t    8007 allocs/op",
            "extra": "2584 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack - ns/op",
            "value": 936279,
            "unit": "ns/op",
            "extra": "2584 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack - MB/s",
            "value": 119.23,
            "unit": "MB/s",
            "extra": "2584 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack - B/op",
            "value": 323891,
            "unit": "B/op",
            "extra": "2584 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack - allocs/op",
            "value": 8007,
            "unit": "allocs/op",
            "extra": "2584 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf",
            "value": 330780,
            "unit": "ns/op\t 122.63 MB/s\t  169202 B/op\t    3466 allocs/op",
            "extra": "7110 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf - ns/op",
            "value": 330780,
            "unit": "ns/op",
            "extra": "7110 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf - MB/s",
            "value": 122.63,
            "unit": "MB/s",
            "extra": "7110 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf - B/op",
            "value": 169202,
            "unit": "B/op",
            "extra": "7110 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf - allocs/op",
            "value": 3466,
            "unit": "allocs/op",
            "extra": "7110 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf",
            "value": 248768,
            "unit": "ns/op\t 163.06 MB/s\t      91 B/op\t       2 allocs/op",
            "extra": "9534 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf - ns/op",
            "value": 248768,
            "unit": "ns/op",
            "extra": "9534 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf - MB/s",
            "value": 163.06,
            "unit": "MB/s",
            "extra": "9534 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf - B/op",
            "value": 91,
            "unit": "B/op",
            "extra": "9534 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "9534 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json",
            "value": 147268,
            "unit": "ns/op\t 252.99 MB/s\t   41078 B/op\t       2 allocs/op",
            "extra": "16297 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json - ns/op",
            "value": 147268,
            "unit": "ns/op",
            "extra": "16297 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json - MB/s",
            "value": 252.99,
            "unit": "MB/s",
            "extra": "16297 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json - B/op",
            "value": 41078,
            "unit": "B/op",
            "extra": "16297 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "16297 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack",
            "value": 104808,
            "unit": "ns/op\t 186.17 MB/s\t   65626 B/op\t      12 allocs/op",
            "extra": "22884 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack - ns/op",
            "value": 104808,
            "unit": "ns/op",
            "extra": "22884 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack - MB/s",
            "value": 186.17,
            "unit": "MB/s",
            "extra": "22884 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack - B/op",
            "value": 65626,
            "unit": "B/op",
            "extra": "22884 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "22884 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf",
            "value": 4463,
            "unit": "ns/op\t1880.32 MB/s\t    9669 B/op\t       3 allocs/op",
            "extra": "546968 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf - ns/op",
            "value": 4463,
            "unit": "ns/op",
            "extra": "546968 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf - MB/s",
            "value": 1880.32,
            "unit": "MB/s",
            "extra": "546968 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf - B/op",
            "value": 9669,
            "unit": "B/op",
            "extra": "546968 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "546968 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json",
            "value": 554809,
            "unit": "ns/op\t  67.15 MB/s\t   54080 B/op\t      40 allocs/op",
            "extra": "4267 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json - ns/op",
            "value": 554809,
            "unit": "ns/op",
            "extra": "4267 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json - MB/s",
            "value": 67.15,
            "unit": "MB/s",
            "extra": "4267 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json - B/op",
            "value": 54080,
            "unit": "B/op",
            "extra": "4267 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json - allocs/op",
            "value": 40,
            "unit": "allocs/op",
            "extra": "4267 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack",
            "value": 139371,
            "unit": "ns/op\t 140.00 MB/s\t   35197 B/op\t      18 allocs/op",
            "extra": "17256 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack - ns/op",
            "value": 139371,
            "unit": "ns/op",
            "extra": "17256 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack - MB/s",
            "value": 140,
            "unit": "MB/s",
            "extra": "17256 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack - B/op",
            "value": 35197,
            "unit": "B/op",
            "extra": "17256 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "17256 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf",
            "value": 4863,
            "unit": "ns/op\t1725.55 MB/s\t   17524 B/op\t       5 allocs/op",
            "extra": "461612 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf - ns/op",
            "value": 4863,
            "unit": "ns/op",
            "extra": "461612 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf - MB/s",
            "value": 1725.55,
            "unit": "MB/s",
            "extra": "461612 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf - B/op",
            "value": 17524,
            "unit": "B/op",
            "extra": "461612 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "461612 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json",
            "value": 123018,
            "unit": "ns/op\t 235.93 MB/s\t   32879 B/op\t       2 allocs/op",
            "extra": "19518 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json - ns/op",
            "value": 123018,
            "unit": "ns/op",
            "extra": "19518 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json - MB/s",
            "value": 235.93,
            "unit": "MB/s",
            "extra": "19518 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json - B/op",
            "value": 32879,
            "unit": "B/op",
            "extra": "19518 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "19518 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack",
            "value": 104098,
            "unit": "ns/op\t 187.51 MB/s\t   65626 B/op\t      12 allocs/op",
            "extra": "22952 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack - ns/op",
            "value": 104098,
            "unit": "ns/op",
            "extra": "22952 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack - MB/s",
            "value": 187.51,
            "unit": "MB/s",
            "extra": "22952 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack - B/op",
            "value": 65626,
            "unit": "B/op",
            "extra": "22952 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "22952 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf",
            "value": 4445,
            "unit": "ns/op\t1889.18 MB/s\t    9669 B/op\t       3 allocs/op",
            "extra": "551166 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf - ns/op",
            "value": 4445,
            "unit": "ns/op",
            "extra": "551166 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf - MB/s",
            "value": 1889.18,
            "unit": "MB/s",
            "extra": "551166 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf - B/op",
            "value": 9669,
            "unit": "B/op",
            "extra": "551166 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "551166 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json",
            "value": 467788,
            "unit": "ns/op\t  62.05 MB/s\t   54088 B/op\t      40 allocs/op",
            "extra": "5036 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json - ns/op",
            "value": 467788,
            "unit": "ns/op",
            "extra": "5036 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json - MB/s",
            "value": 62.05,
            "unit": "MB/s",
            "extra": "5036 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json - B/op",
            "value": 54088,
            "unit": "B/op",
            "extra": "5036 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json - allocs/op",
            "value": 40,
            "unit": "allocs/op",
            "extra": "5036 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack",
            "value": 139407,
            "unit": "ns/op\t 140.01 MB/s\t   35205 B/op\t      18 allocs/op",
            "extra": "17200 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack - ns/op",
            "value": 139407,
            "unit": "ns/op",
            "extra": "17200 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack - MB/s",
            "value": 140.01,
            "unit": "MB/s",
            "extra": "17200 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack - B/op",
            "value": 35205,
            "unit": "B/op",
            "extra": "17200 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "17200 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf",
            "value": 4826,
            "unit": "ns/op\t1740.13 MB/s\t   17532 B/op\t       5 allocs/op",
            "extra": "469960 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf - ns/op",
            "value": 4826,
            "unit": "ns/op",
            "extra": "469960 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf - MB/s",
            "value": 1740.13,
            "unit": "MB/s",
            "extra": "469960 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf - B/op",
            "value": 17532,
            "unit": "B/op",
            "extra": "469960 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "469960 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json",
            "value": 121627,
            "unit": "ns/op\t 238.63 MB/s\t   32875 B/op\t       2 allocs/op",
            "extra": "19731 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json - ns/op",
            "value": 121627,
            "unit": "ns/op",
            "extra": "19731 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json - MB/s",
            "value": 238.63,
            "unit": "MB/s",
            "extra": "19731 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json - B/op",
            "value": 32875,
            "unit": "B/op",
            "extra": "19731 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "19731 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack",
            "value": 103590,
            "unit": "ns/op\t 188.43 MB/s\t   65626 B/op\t      12 allocs/op",
            "extra": "23160 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack - ns/op",
            "value": 103590,
            "unit": "ns/op",
            "extra": "23160 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack - MB/s",
            "value": 188.43,
            "unit": "MB/s",
            "extra": "23160 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack - B/op",
            "value": 65626,
            "unit": "B/op",
            "extra": "23160 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "23160 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf",
            "value": 49025,
            "unit": "ns/op\t  47.06 MB/s\t   11324 B/op\t      14 allocs/op",
            "extra": "49016 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf - ns/op",
            "value": 49025,
            "unit": "ns/op",
            "extra": "49016 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf - MB/s",
            "value": 47.06,
            "unit": "MB/s",
            "extra": "49016 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf - B/op",
            "value": 11324,
            "unit": "B/op",
            "extra": "49016 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf - allocs/op",
            "value": 14,
            "unit": "allocs/op",
            "extra": "49016 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json",
            "value": 467283,
            "unit": "ns/op\t  62.11 MB/s\t   54088 B/op\t      40 allocs/op",
            "extra": "5022 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json - ns/op",
            "value": 467283,
            "unit": "ns/op",
            "extra": "5022 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json - MB/s",
            "value": 62.11,
            "unit": "MB/s",
            "extra": "5022 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json - B/op",
            "value": 54088,
            "unit": "B/op",
            "extra": "5022 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json - allocs/op",
            "value": 40,
            "unit": "allocs/op",
            "extra": "5022 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack",
            "value": 141012,
            "unit": "ns/op\t 138.42 MB/s\t   35205 B/op\t      18 allocs/op",
            "extra": "17185 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack - ns/op",
            "value": 141012,
            "unit": "ns/op",
            "extra": "17185 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack - MB/s",
            "value": 138.42,
            "unit": "MB/s",
            "extra": "17185 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack - B/op",
            "value": 35205,
            "unit": "B/op",
            "extra": "17185 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "17185 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf",
            "value": 43572,
            "unit": "ns/op\t  52.95 MB/s\t   17611 B/op\t       7 allocs/op",
            "extra": "54847 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf - ns/op",
            "value": 43572,
            "unit": "ns/op",
            "extra": "54847 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf - MB/s",
            "value": 52.95,
            "unit": "MB/s",
            "extra": "54847 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf - B/op",
            "value": 17611,
            "unit": "B/op",
            "extra": "54847 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "54847 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json",
            "value": 22932,
            "unit": "ns/op\t 180.36 MB/s\t    4913 B/op\t       2 allocs/op",
            "extra": "104167 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json - ns/op",
            "value": 22932,
            "unit": "ns/op",
            "extra": "104167 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json - MB/s",
            "value": 180.36,
            "unit": "MB/s",
            "extra": "104167 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json - B/op",
            "value": 4913,
            "unit": "B/op",
            "extra": "104167 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "104167 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack",
            "value": 41370,
            "unit": "ns/op\t 223.62 MB/s\t   32805 B/op\t      11 allocs/op",
            "extra": "57661 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack - ns/op",
            "value": 41370,
            "unit": "ns/op",
            "extra": "57661 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack - MB/s",
            "value": 223.62,
            "unit": "MB/s",
            "extra": "57661 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack - B/op",
            "value": 32805,
            "unit": "B/op",
            "extra": "57661 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "57661 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf",
            "value": 4884,
            "unit": "ns/op\t  20.27 MB/s\t     208 B/op\t       3 allocs/op",
            "extra": "488998 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf - ns/op",
            "value": 4884,
            "unit": "ns/op",
            "extra": "488998 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf - MB/s",
            "value": 20.27,
            "unit": "MB/s",
            "extra": "488998 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf - B/op",
            "value": 208,
            "unit": "B/op",
            "extra": "488998 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "488998 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json",
            "value": 121676,
            "unit": "ns/op\t  33.99 MB/s\t   25504 B/op\t      19 allocs/op",
            "extra": "19765 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json - ns/op",
            "value": 121676,
            "unit": "ns/op",
            "extra": "19765 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json - MB/s",
            "value": 33.99,
            "unit": "MB/s",
            "extra": "19765 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json - B/op",
            "value": 25504,
            "unit": "B/op",
            "extra": "19765 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json - allocs/op",
            "value": 19,
            "unit": "allocs/op",
            "extra": "19765 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack",
            "value": 54521,
            "unit": "ns/op\t 169.68 MB/s\t   16570 B/op\t       8 allocs/op",
            "extra": "44100 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack - ns/op",
            "value": 54521,
            "unit": "ns/op",
            "extra": "44100 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack - MB/s",
            "value": 169.68,
            "unit": "MB/s",
            "extra": "44100 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack - B/op",
            "value": 16570,
            "unit": "B/op",
            "extra": "44100 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "44100 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf",
            "value": 2433,
            "unit": "ns/op\t  40.69 MB/s\t    8258 B/op\t       3 allocs/op",
            "extra": "995185 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf - ns/op",
            "value": 2433,
            "unit": "ns/op",
            "extra": "995185 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf - MB/s",
            "value": 40.69,
            "unit": "MB/s",
            "extra": "995185 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf - B/op",
            "value": 8258,
            "unit": "B/op",
            "extra": "995185 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "995185 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json",
            "value": 57467,
            "unit": "ns/op\t 145.91 MB/s\t    9525 B/op\t       2 allocs/op",
            "extra": "41607 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json - ns/op",
            "value": 57467,
            "unit": "ns/op",
            "extra": "41607 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json - MB/s",
            "value": 145.91,
            "unit": "MB/s",
            "extra": "41607 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json - B/op",
            "value": 9525,
            "unit": "B/op",
            "extra": "41607 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "41607 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack",
            "value": 26309,
            "unit": "ns/op\t 146.87 MB/s\t    8225 B/op\t       9 allocs/op",
            "extra": "90092 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack - ns/op",
            "value": 26309,
            "unit": "ns/op",
            "extra": "90092 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack - MB/s",
            "value": 146.87,
            "unit": "MB/s",
            "extra": "90092 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack - B/op",
            "value": 8225,
            "unit": "B/op",
            "extra": "90092 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack - allocs/op",
            "value": 9,
            "unit": "allocs/op",
            "extra": "90092 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf",
            "value": 959.7,
            "unit": "ns/op\t3233.16 MB/s\t    3297 B/op\t       3 allocs/op",
            "extra": "2515318 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf - ns/op",
            "value": 959.7,
            "unit": "ns/op",
            "extra": "2515318 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf - MB/s",
            "value": 3233.16,
            "unit": "MB/s",
            "extra": "2515318 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf - B/op",
            "value": 3297,
            "unit": "B/op",
            "extra": "2515318 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "2515318 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json",
            "value": 162254,
            "unit": "ns/op\t  51.68 MB/s\t    7832 B/op\t      17 allocs/op",
            "extra": "14787 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json - ns/op",
            "value": 162254,
            "unit": "ns/op",
            "extra": "14787 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json - MB/s",
            "value": 51.68,
            "unit": "MB/s",
            "extra": "14787 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json - B/op",
            "value": 7832,
            "unit": "B/op",
            "extra": "14787 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "14787 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack",
            "value": 38684,
            "unit": "ns/op\t  99.89 MB/s\t    6320 B/op\t       8 allocs/op",
            "extra": "62232 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack - ns/op",
            "value": 38684,
            "unit": "ns/op",
            "extra": "62232 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack - MB/s",
            "value": 99.89,
            "unit": "MB/s",
            "extra": "62232 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack - B/op",
            "value": 6320,
            "unit": "B/op",
            "extra": "62232 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "62232 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf",
            "value": 837.9,
            "unit": "ns/op\t3703.46 MB/s\t    3129 B/op\t       3 allocs/op",
            "extra": "2880409 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf - ns/op",
            "value": 837.9,
            "unit": "ns/op",
            "extra": "2880409 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf - MB/s",
            "value": 3703.46,
            "unit": "MB/s",
            "extra": "2880409 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf - B/op",
            "value": 3129,
            "unit": "B/op",
            "extra": "2880409 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "2880409 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json",
            "value": 1975,
            "unit": "ns/op\t 126.57 MB/s\t     936 B/op\t      22 allocs/op",
            "extra": "1214620 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json - ns/op",
            "value": 1975,
            "unit": "ns/op",
            "extra": "1214620 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json - MB/s",
            "value": 126.57,
            "unit": "MB/s",
            "extra": "1214620 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json - B/op",
            "value": 936,
            "unit": "B/op",
            "extra": "1214620 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json - allocs/op",
            "value": 22,
            "unit": "allocs/op",
            "extra": "1214620 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack",
            "value": 1603,
            "unit": "ns/op\t 122.86 MB/s\t     680 B/op\t      15 allocs/op",
            "extra": "1497528 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack - ns/op",
            "value": 1603,
            "unit": "ns/op",
            "extra": "1497528 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack - MB/s",
            "value": 122.86,
            "unit": "MB/s",
            "extra": "1497528 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack - B/op",
            "value": 680,
            "unit": "B/op",
            "extra": "1497528 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "1497528 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf",
            "value": 1262,
            "unit": "ns/op\t 178.25 MB/s\t     368 B/op\t       3 allocs/op",
            "extra": "1899320 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf - ns/op",
            "value": 1262,
            "unit": "ns/op",
            "extra": "1899320 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf - MB/s",
            "value": 178.25,
            "unit": "MB/s",
            "extra": "1899320 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf - B/op",
            "value": 368,
            "unit": "B/op",
            "extra": "1899320 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1899320 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json",
            "value": 5723,
            "unit": "ns/op\t  43.68 MB/s\t    1352 B/op\t      41 allocs/op",
            "extra": "416762 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json - ns/op",
            "value": 5723,
            "unit": "ns/op",
            "extra": "416762 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json - MB/s",
            "value": 43.68,
            "unit": "MB/s",
            "extra": "416762 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json - B/op",
            "value": 1352,
            "unit": "B/op",
            "extra": "416762 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json - allocs/op",
            "value": 41,
            "unit": "allocs/op",
            "extra": "416762 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack",
            "value": 2547,
            "unit": "ns/op\t  77.34 MB/s\t    1064 B/op\t      34 allocs/op",
            "extra": "913928 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack - ns/op",
            "value": 2547,
            "unit": "ns/op",
            "extra": "913928 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack - MB/s",
            "value": 77.34,
            "unit": "MB/s",
            "extra": "913928 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack - B/op",
            "value": 1064,
            "unit": "B/op",
            "extra": "913928 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack - allocs/op",
            "value": 34,
            "unit": "allocs/op",
            "extra": "913928 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf",
            "value": 1520,
            "unit": "ns/op\t 148.00 MB/s\t     895 B/op\t      17 allocs/op",
            "extra": "1591034 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf - ns/op",
            "value": 1520,
            "unit": "ns/op",
            "extra": "1591034 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf - MB/s",
            "value": 148,
            "unit": "MB/s",
            "extra": "1591034 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf - B/op",
            "value": 895,
            "unit": "B/op",
            "extra": "1591034 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "1591034 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json",
            "value": 1765038,
            "unit": "ns/op\t 404.97 MB/s\t  724310 B/op\t       3 allocs/op",
            "extra": "1329 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json - ns/op",
            "value": 1765038,
            "unit": "ns/op",
            "extra": "1329 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json - MB/s",
            "value": 404.97,
            "unit": "MB/s",
            "extra": "1329 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json - B/op",
            "value": 724310,
            "unit": "B/op",
            "extra": "1329 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1329 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack",
            "value": 2547660,
            "unit": "ns/op\t 219.22 MB/s\t 2217446 B/op\t    5018 allocs/op",
            "extra": "896 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack - ns/op",
            "value": 2547660,
            "unit": "ns/op",
            "extra": "896 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack - MB/s",
            "value": 219.22,
            "unit": "MB/s",
            "extra": "896 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack - B/op",
            "value": 2217446,
            "unit": "B/op",
            "extra": "896 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack - allocs/op",
            "value": 5018,
            "unit": "allocs/op",
            "extra": "896 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf",
            "value": 1803729,
            "unit": "ns/op\t 106.58 MB/s\t 1518598 B/op\t      51 allocs/op",
            "extra": "1357 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf - ns/op",
            "value": 1803729,
            "unit": "ns/op",
            "extra": "1357 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf - MB/s",
            "value": 106.58,
            "unit": "MB/s",
            "extra": "1357 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf - B/op",
            "value": 1518598,
            "unit": "B/op",
            "extra": "1357 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "1357 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json",
            "value": 13295239,
            "unit": "ns/op\t  53.76 MB/s\t 3074430 B/op\t   45025 allocs/op",
            "extra": "180 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json - ns/op",
            "value": 13295239,
            "unit": "ns/op",
            "extra": "180 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json - MB/s",
            "value": 53.76,
            "unit": "MB/s",
            "extra": "180 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json - B/op",
            "value": 3074430,
            "unit": "B/op",
            "extra": "180 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json - allocs/op",
            "value": 45025,
            "unit": "allocs/op",
            "extra": "180 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack",
            "value": 4916231,
            "unit": "ns/op\t 113.61 MB/s\t 1602203 B/op\t   40008 allocs/op",
            "extra": "484 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack - ns/op",
            "value": 4916231,
            "unit": "ns/op",
            "extra": "484 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack - MB/s",
            "value": 113.61,
            "unit": "MB/s",
            "extra": "484 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack - B/op",
            "value": 1602203,
            "unit": "B/op",
            "extra": "484 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack - allocs/op",
            "value": 40008,
            "unit": "allocs/op",
            "extra": "484 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf",
            "value": 2379923,
            "unit": "ns/op\t  80.77 MB/s\t 1823005 B/op\t   16296 allocs/op",
            "extra": "1020 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf - ns/op",
            "value": 2379923,
            "unit": "ns/op",
            "extra": "1020 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf - MB/s",
            "value": 80.77,
            "unit": "MB/s",
            "extra": "1020 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf - B/op",
            "value": 1823005,
            "unit": "B/op",
            "extra": "1020 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf - allocs/op",
            "value": 16296,
            "unit": "allocs/op",
            "extra": "1020 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/json",
            "value": 45528,
            "unit": "ns/op\t   10980 B/op\t       2 allocs/op",
            "extra": "52534 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/json - ns/op",
            "value": 45528,
            "unit": "ns/op",
            "extra": "52534 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/json - B/op",
            "value": 10980,
            "unit": "B/op",
            "extra": "52534 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "52534 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/msgpack",
            "value": 55690,
            "unit": "ns/op\t   32852 B/op\t      11 allocs/op",
            "extra": "43028 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/msgpack - ns/op",
            "value": 55690,
            "unit": "ns/op",
            "extra": "43028 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/msgpack - B/op",
            "value": 32852,
            "unit": "B/op",
            "extra": "43028 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/msgpack - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "43028 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_fast",
            "value": 7534,
            "unit": "ns/op\t    6979 B/op\t       3 allocs/op",
            "extra": "318116 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_fast - ns/op",
            "value": 7534,
            "unit": "ns/op",
            "extra": "318116 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_fast - B/op",
            "value": 6979,
            "unit": "B/op",
            "extra": "318116 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "318116 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_qpack",
            "value": 2755,
            "unit": "ns/op\t    2496 B/op\t       3 allocs/op",
            "extra": "865520 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_qpack - ns/op",
            "value": 2755,
            "unit": "ns/op",
            "extra": "865520 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_qpack - B/op",
            "value": 2496,
            "unit": "B/op",
            "extra": "865520 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_qpack - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "865520 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_dense",
            "value": 2878,
            "unit": "ns/op\t    2496 B/op\t       3 allocs/op",
            "extra": "805303 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_dense - ns/op",
            "value": 2878,
            "unit": "ns/op",
            "extra": "805303 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_dense - B/op",
            "value": 2496,
            "unit": "B/op",
            "extra": "805303 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "805303 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/json",
            "value": 209332,
            "unit": "ns/op\t   21288 B/op\t      41 allocs/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/json - ns/op",
            "value": 209332,
            "unit": "ns/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/json - B/op",
            "value": 21288,
            "unit": "B/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/json - allocs/op",
            "value": 41,
            "unit": "allocs/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/msgpack",
            "value": 77640,
            "unit": "ns/op\t   21427 B/op\t      22 allocs/op",
            "extra": "30828 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/msgpack - ns/op",
            "value": 77640,
            "unit": "ns/op",
            "extra": "30828 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/msgpack - B/op",
            "value": 21427,
            "unit": "B/op",
            "extra": "30828 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/msgpack - allocs/op",
            "value": 22,
            "unit": "allocs/op",
            "extra": "30828 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_fast",
            "value": 13690,
            "unit": "ns/op\t   10594 B/op\t       5 allocs/op",
            "extra": "174938 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_fast - ns/op",
            "value": 13690,
            "unit": "ns/op",
            "extra": "174938 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_fast - B/op",
            "value": 10594,
            "unit": "B/op",
            "extra": "174938 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_fast - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "174938 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_qpack",
            "value": 3074,
            "unit": "ns/op\t   10595 B/op\t       5 allocs/op",
            "extra": "804228 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_qpack - ns/op",
            "value": 3074,
            "unit": "ns/op",
            "extra": "804228 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_qpack - B/op",
            "value": 10595,
            "unit": "B/op",
            "extra": "804228 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_qpack - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "804228 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_dense",
            "value": 3380,
            "unit": "ns/op\t   10676 B/op\t       7 allocs/op",
            "extra": "738328 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_dense - ns/op",
            "value": 3380,
            "unit": "ns/op",
            "extra": "738328 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_dense - B/op",
            "value": 10676,
            "unit": "B/op",
            "extra": "738328 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_dense - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "738328 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/json",
            "value": 3651,
            "unit": "ns/op\t     488 B/op\t      12 allocs/op",
            "extra": "633568 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/json - ns/op",
            "value": 3651,
            "unit": "ns/op",
            "extra": "633568 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/json - B/op",
            "value": 488,
            "unit": "B/op",
            "extra": "633568 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/json - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "633568 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/msgpack",
            "value": 1168,
            "unit": "ns/op\t     312 B/op\t       9 allocs/op",
            "extra": "2042640 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/msgpack - ns/op",
            "value": 1168,
            "unit": "ns/op",
            "extra": "2042640 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/msgpack - B/op",
            "value": 312,
            "unit": "B/op",
            "extra": "2042640 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/msgpack - allocs/op",
            "value": 9,
            "unit": "allocs/op",
            "extra": "2042640 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/qdf_fast",
            "value": 533.4,
            "unit": "ns/op\t     264 B/op\t       8 allocs/op",
            "extra": "4503618 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/qdf_fast - ns/op",
            "value": 533.4,
            "unit": "ns/op",
            "extra": "4503618 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/qdf_fast - B/op",
            "value": 264,
            "unit": "B/op",
            "extra": "4503618 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/qdf_fast - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "4503618 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/json",
            "value": 1629,
            "unit": "ns/op\t     488 B/op\t      12 allocs/op",
            "extra": "1472829 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/json - ns/op",
            "value": 1629,
            "unit": "ns/op",
            "extra": "1472829 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/json - B/op",
            "value": 488,
            "unit": "B/op",
            "extra": "1472829 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/json - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "1472829 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/msgpack",
            "value": 551.6,
            "unit": "ns/op\t     312 B/op\t       9 allocs/op",
            "extra": "4355113 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/msgpack - ns/op",
            "value": 551.6,
            "unit": "ns/op",
            "extra": "4355113 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/msgpack - B/op",
            "value": 312,
            "unit": "B/op",
            "extra": "4355113 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/msgpack - allocs/op",
            "value": 9,
            "unit": "allocs/op",
            "extra": "4355113 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/qdf_fast",
            "value": 271.6,
            "unit": "ns/op\t     264 B/op\t       8 allocs/op",
            "extra": "8812273 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/qdf_fast - ns/op",
            "value": 271.6,
            "unit": "ns/op",
            "extra": "8812273 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/qdf_fast - B/op",
            "value": 264,
            "unit": "B/op",
            "extra": "8812273 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/qdf_fast - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "8812273 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense",
            "value": 365504,
            "unit": "ns/op\t 508.04 MB/s\t  376005 B/op\t    2011 allocs/op",
            "extra": "6344 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense - ns/op",
            "value": 365504,
            "unit": "ns/op",
            "extra": "6344 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense - MB/s",
            "value": 508.04,
            "unit": "MB/s",
            "extra": "6344 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense - B/op",
            "value": 376005,
            "unit": "B/op",
            "extra": "6344 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense - allocs/op",
            "value": 2011,
            "unit": "allocs/op",
            "extra": "6344 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/json",
            "value": 2225,
            "unit": "ns/op\t     800 B/op\t      14 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/json - ns/op",
            "value": 2225,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/json - B/op",
            "value": 800,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/json - allocs/op",
            "value": 14,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/msgpack",
            "value": 1924,
            "unit": "ns/op\t    1008 B/op\t      17 allocs/op",
            "extra": "1246398 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/msgpack - ns/op",
            "value": 1924,
            "unit": "ns/op",
            "extra": "1246398 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/msgpack - B/op",
            "value": 1008,
            "unit": "B/op",
            "extra": "1246398 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/msgpack - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "1246398 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/qdf_fast",
            "value": 1549,
            "unit": "ns/op\t     696 B/op\t      13 allocs/op",
            "extra": "1551615 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/qdf_fast - ns/op",
            "value": 1549,
            "unit": "ns/op",
            "extra": "1551615 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/qdf_fast - B/op",
            "value": 696,
            "unit": "B/op",
            "extra": "1551615 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/qdf_fast - allocs/op",
            "value": 13,
            "unit": "allocs/op",
            "extra": "1551615 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/json",
            "value": 1034,
            "unit": "ns/op\t     364 B/op\t       5 allocs/op",
            "extra": "2319541 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/json - ns/op",
            "value": 1034,
            "unit": "ns/op",
            "extra": "2319541 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/json - B/op",
            "value": 364,
            "unit": "B/op",
            "extra": "2319541 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/json - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "2319541 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/msgpack",
            "value": 1026,
            "unit": "ns/op\t     538 B/op\t       7 allocs/op",
            "extra": "2339895 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/msgpack - ns/op",
            "value": 1026,
            "unit": "ns/op",
            "extra": "2339895 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/msgpack - B/op",
            "value": 538,
            "unit": "B/op",
            "extra": "2339895 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/msgpack - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "2339895 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/qdf_fast",
            "value": 785.5,
            "unit": "ns/op\t     388 B/op\t       5 allocs/op",
            "extra": "3056259 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/qdf_fast - ns/op",
            "value": 785.5,
            "unit": "ns/op",
            "extra": "3056259 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/qdf_fast - B/op",
            "value": 388,
            "unit": "B/op",
            "extra": "3056259 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/qdf_fast - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "3056259 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/json",
            "value": 288175,
            "unit": "ns/op\t  122924 B/op\t       3 allocs/op",
            "extra": "7832 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/json - ns/op",
            "value": 288175,
            "unit": "ns/op",
            "extra": "7832 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/json - B/op",
            "value": 122924,
            "unit": "B/op",
            "extra": "7832 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/json - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "7832 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/msgpack",
            "value": 245060,
            "unit": "ns/op\t  189946 B/op\t      10 allocs/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/msgpack - ns/op",
            "value": 245060,
            "unit": "ns/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/msgpack - B/op",
            "value": 189946,
            "unit": "B/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/msgpack - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/qdf_fast",
            "value": 89674,
            "unit": "ns/op\t   91149 B/op\t       4 allocs/op",
            "extra": "26653 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/qdf_fast - ns/op",
            "value": 89674,
            "unit": "ns/op",
            "extra": "26653 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/qdf_fast - B/op",
            "value": 91149,
            "unit": "B/op",
            "extra": "26653 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/qdf_fast - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "26653 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/json",
            "value": 1084,
            "unit": "ns/op\t     800 B/op\t      14 allocs/op",
            "extra": "2193420 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/json - ns/op",
            "value": 1084,
            "unit": "ns/op",
            "extra": "2193420 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/json - B/op",
            "value": 800,
            "unit": "B/op",
            "extra": "2193420 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/json - allocs/op",
            "value": 14,
            "unit": "allocs/op",
            "extra": "2193420 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/msgpack",
            "value": 986.6,
            "unit": "ns/op\t    1008 B/op\t      17 allocs/op",
            "extra": "2452302 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/msgpack - ns/op",
            "value": 986.6,
            "unit": "ns/op",
            "extra": "2452302 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/msgpack - B/op",
            "value": 1008,
            "unit": "B/op",
            "extra": "2452302 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/msgpack - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "2452302 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/qdf_fast",
            "value": 788.6,
            "unit": "ns/op\t     697 B/op\t      13 allocs/op",
            "extra": "3018726 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/qdf_fast - ns/op",
            "value": 788.6,
            "unit": "ns/op",
            "extra": "3018726 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/qdf_fast - B/op",
            "value": 697,
            "unit": "B/op",
            "extra": "3018726 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/qdf_fast - allocs/op",
            "value": 13,
            "unit": "allocs/op",
            "extra": "3018726 times\n4 procs"
          }
        ]
      }
    ]
  }
}
window.BENCHMARK_DATA = {
  "lastUpdate": 1780054581490,
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
      },
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
          "id": "d4bcfb0e4aa0ba8360de3385f91b88825768613b",
          "message": "docs: plain-language qdf_simd guide + sync SIMD coverage\n\nAdds a 'when to use the qdf_simd build tag' section to docs/USAGE.md aimed\nat app devs: what it accelerates (QPack integer/bool codecs), which data\nshapes benefit (large numeric/bool slices) vs not (small/string/struct-heavy,\nfloat codecs), supported architectures (amd64+AVX2, runtime CPUID fallback,\nscalar stub elsewhere, no GOEXPERIMENT needed), and a concrete recommendation.\n\nAlso fixes stale SIMD docs: GUIDE and README claimed decode-only bit-unpack\nat {8,16,32} and 'encode-side SIMD not yet implemented'. Now reflects what\nships — encode VPSHUFB {8,16,32}, decode {8,16,32} + {10,12,14,20}, []bool\npack — and notes output is byte-identical to scalar.",
          "timestamp": "2026-05-29T13:25:12+03:00",
          "tree_id": "cda6cbbd2fa135d73a582c4b9d1f1bc8eda9eebb",
          "url": "https://github.com/alex60217101990/qdf/commit/d4bcfb0e4aa0ba8360de3385f91b88825768613b"
        },
        "date": 1780050810309,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json",
            "value": 163.4,
            "unit": "ns/op\t 153.04 MB/s\t      24 B/op\t       1 allocs/op",
            "extra": "14785884 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json - ns/op",
            "value": 163.4,
            "unit": "ns/op",
            "extra": "14785884 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json - MB/s",
            "value": 153.04,
            "unit": "MB/s",
            "extra": "14785884 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json - B/op",
            "value": 24,
            "unit": "B/op",
            "extra": "14785884 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "14785884 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal",
            "value": 189.5,
            "unit": "ns/op\t 126.67 MB/s\t      48 B/op\t       2 allocs/op",
            "extra": "12629578 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal - ns/op",
            "value": 189.5,
            "unit": "ns/op",
            "extra": "12629578 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal - MB/s",
            "value": 126.67,
            "unit": "MB/s",
            "extra": "12629578 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal - B/op",
            "value": 48,
            "unit": "B/op",
            "extra": "12629578 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "12629578 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack",
            "value": 241.9,
            "unit": "ns/op\t  66.14 MB/s\t     136 B/op\t       3 allocs/op",
            "extra": "9685370 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack - ns/op",
            "value": 241.9,
            "unit": "ns/op",
            "extra": "9685370 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack - MB/s",
            "value": 66.14,
            "unit": "MB/s",
            "extra": "9685370 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack - B/op",
            "value": 136,
            "unit": "B/op",
            "extra": "9685370 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "9685370 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast",
            "value": 312.1,
            "unit": "ns/op\t  70.48 MB/s\t      72 B/op\t       3 allocs/op",
            "extra": "7639310 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast - ns/op",
            "value": 312.1,
            "unit": "ns/op",
            "extra": "7639310 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast - MB/s",
            "value": 70.48,
            "unit": "MB/s",
            "extra": "7639310 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast - B/op",
            "value": 72,
            "unit": "B/op",
            "extra": "7639310 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "7639310 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense",
            "value": 390.4,
            "unit": "ns/op\t  64.03 MB/s\t      80 B/op\t       3 allocs/op",
            "extra": "6153765 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense - ns/op",
            "value": 390.4,
            "unit": "ns/op",
            "extra": "6153765 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense - MB/s",
            "value": 64.03,
            "unit": "MB/s",
            "extra": "6153765 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "6153765 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "6153765 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json",
            "value": 1013,
            "unit": "ns/op\t 208.34 MB/s\t     192 B/op\t       1 allocs/op",
            "extra": "2367775 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json - ns/op",
            "value": 1013,
            "unit": "ns/op",
            "extra": "2367775 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json - MB/s",
            "value": 208.34,
            "unit": "MB/s",
            "extra": "2367775 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json - B/op",
            "value": 192,
            "unit": "B/op",
            "extra": "2367775 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2367775 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal",
            "value": 1058,
            "unit": "ns/op\t 198.41 MB/s\t     416 B/op\t       2 allocs/op",
            "extra": "2267283 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal - ns/op",
            "value": 1058,
            "unit": "ns/op",
            "extra": "2267283 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal - MB/s",
            "value": 198.41,
            "unit": "MB/s",
            "extra": "2267283 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal - B/op",
            "value": 416,
            "unit": "B/op",
            "extra": "2267283 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "2267283 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack",
            "value": 1010,
            "unit": "ns/op\t 132.67 MB/s\t     688 B/op\t       5 allocs/op",
            "extra": "2374743 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack - ns/op",
            "value": 1010,
            "unit": "ns/op",
            "extra": "2374743 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack - MB/s",
            "value": 132.67,
            "unit": "MB/s",
            "extra": "2374743 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack - B/op",
            "value": 688,
            "unit": "B/op",
            "extra": "2374743 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "2374743 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast",
            "value": 601.7,
            "unit": "ns/op\t 219.38 MB/s\t     528 B/op\t       3 allocs/op",
            "extra": "3823363 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast - ns/op",
            "value": 601.7,
            "unit": "ns/op",
            "extra": "3823363 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast - MB/s",
            "value": 219.38,
            "unit": "MB/s",
            "extra": "3823363 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast - B/op",
            "value": 528,
            "unit": "B/op",
            "extra": "3823363 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3823363 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense",
            "value": 766.8,
            "unit": "ns/op\t 179.98 MB/s\t     528 B/op\t       3 allocs/op",
            "extra": "3120639 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense - ns/op",
            "value": 766.8,
            "unit": "ns/op",
            "extra": "3120639 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense - MB/s",
            "value": 179.98,
            "unit": "MB/s",
            "extra": "3120639 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense - B/op",
            "value": 528,
            "unit": "B/op",
            "extra": "3120639 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3120639 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json",
            "value": 464.1,
            "unit": "ns/op\t 224.11 MB/s\t      80 B/op\t       1 allocs/op",
            "extra": "5257680 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json - ns/op",
            "value": 464.1,
            "unit": "ns/op",
            "extra": "5257680 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json - MB/s",
            "value": 224.11,
            "unit": "MB/s",
            "extra": "5257680 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "5257680 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "5257680 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal",
            "value": 481.9,
            "unit": "ns/op\t 213.74 MB/s\t     192 B/op\t       2 allocs/op",
            "extra": "4965973 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal - ns/op",
            "value": 481.9,
            "unit": "ns/op",
            "extra": "4965973 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal - MB/s",
            "value": 213.74,
            "unit": "MB/s",
            "extra": "4965973 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal - B/op",
            "value": 192,
            "unit": "B/op",
            "extra": "4965973 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "4965973 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack",
            "value": 737.4,
            "unit": "ns/op\t 103.07 MB/s\t     320 B/op\t       4 allocs/op",
            "extra": "3257197 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack - ns/op",
            "value": 737.4,
            "unit": "ns/op",
            "extra": "3257197 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack - MB/s",
            "value": 103.07,
            "unit": "MB/s",
            "extra": "3257197 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack - B/op",
            "value": 320,
            "unit": "B/op",
            "extra": "3257197 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "3257197 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast",
            "value": 444.4,
            "unit": "ns/op\t 193.52 MB/s\t     256 B/op\t       3 allocs/op",
            "extra": "5366064 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast - ns/op",
            "value": 444.4,
            "unit": "ns/op",
            "extra": "5366064 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast - MB/s",
            "value": 193.52,
            "unit": "MB/s",
            "extra": "5366064 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast - B/op",
            "value": 256,
            "unit": "B/op",
            "extra": "5366064 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "5366064 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense",
            "value": 656.4,
            "unit": "ns/op\t 146.26 MB/s\t     256 B/op\t       3 allocs/op",
            "extra": "3675162 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense - ns/op",
            "value": 656.4,
            "unit": "ns/op",
            "extra": "3675162 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense - MB/s",
            "value": 146.26,
            "unit": "MB/s",
            "extra": "3675162 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense - B/op",
            "value": 256,
            "unit": "B/op",
            "extra": "3675162 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3675162 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json",
            "value": 1462,
            "unit": "ns/op\t 164.18 MB/s\t       0 B/op\t       0 allocs/op",
            "extra": "1645012 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json - ns/op",
            "value": 1462,
            "unit": "ns/op",
            "extra": "1645012 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json - MB/s",
            "value": 164.18,
            "unit": "MB/s",
            "extra": "1645012 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1645012 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1645012 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal",
            "value": 1495,
            "unit": "ns/op\t 159.89 MB/s\t     240 B/op\t       1 allocs/op",
            "extra": "1605388 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal - ns/op",
            "value": 1495,
            "unit": "ns/op",
            "extra": "1605388 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal - MB/s",
            "value": 159.89,
            "unit": "MB/s",
            "extra": "1605388 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal - B/op",
            "value": 240,
            "unit": "B/op",
            "extra": "1605388 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "1605388 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack",
            "value": 2901,
            "unit": "ns/op\t  47.91 MB/s\t     752 B/op\t      20 allocs/op",
            "extra": "760021 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack - ns/op",
            "value": 2901,
            "unit": "ns/op",
            "extra": "760021 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack - MB/s",
            "value": 47.91,
            "unit": "MB/s",
            "extra": "760021 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack - B/op",
            "value": 752,
            "unit": "B/op",
            "extra": "760021 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "760021 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast",
            "value": 751.3,
            "unit": "ns/op\t 220.95 MB/s\t     176 B/op\t       1 allocs/op",
            "extra": "3182647 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast - ns/op",
            "value": 751.3,
            "unit": "ns/op",
            "extra": "3182647 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast - MB/s",
            "value": 220.95,
            "unit": "MB/s",
            "extra": "3182647 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast - B/op",
            "value": 176,
            "unit": "B/op",
            "extra": "3182647 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "3182647 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense",
            "value": 681.5,
            "unit": "ns/op\t  92.44 MB/s\t      64 B/op\t       1 allocs/op",
            "extra": "3529576 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense - ns/op",
            "value": 681.5,
            "unit": "ns/op",
            "extra": "3529576 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense - MB/s",
            "value": 92.44,
            "unit": "MB/s",
            "extra": "3529576 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense - B/op",
            "value": 64,
            "unit": "B/op",
            "extra": "3529576 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "3529576 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json",
            "value": 870239,
            "unit": "ns/op\t 244.65 MB/s\t     291 B/op\t       1 allocs/op",
            "extra": "2767 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json - ns/op",
            "value": 870239,
            "unit": "ns/op",
            "extra": "2767 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json - MB/s",
            "value": 244.65,
            "unit": "MB/s",
            "extra": "2767 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json - B/op",
            "value": 291,
            "unit": "B/op",
            "extra": "2767 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2767 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal",
            "value": 891908,
            "unit": "ns/op\t 238.70 MB/s\t  213445 B/op\t       2 allocs/op",
            "extra": "2642 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal - ns/op",
            "value": 891908,
            "unit": "ns/op",
            "extra": "2642 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal - MB/s",
            "value": 238.7,
            "unit": "MB/s",
            "extra": "2642 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal - B/op",
            "value": 213445,
            "unit": "B/op",
            "extra": "2642 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "2642 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack",
            "value": 768831,
            "unit": "ns/op\t 176.41 MB/s\t  524379 B/op\t      15 allocs/op",
            "extra": "3093 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack - ns/op",
            "value": 768831,
            "unit": "ns/op",
            "extra": "3093 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack - MB/s",
            "value": 176.41,
            "unit": "MB/s",
            "extra": "3093 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack - B/op",
            "value": 524379,
            "unit": "B/op",
            "extra": "3093 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "3093 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast",
            "value": 239524,
            "unit": "ns/op\t 537.03 MB/s\t  131157 B/op\t       3 allocs/op",
            "extra": "9692 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast - ns/op",
            "value": 239524,
            "unit": "ns/op",
            "extra": "9692 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast - MB/s",
            "value": 537.03,
            "unit": "MB/s",
            "extra": "9692 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast - B/op",
            "value": 131157,
            "unit": "B/op",
            "extra": "9692 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "9692 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense",
            "value": 160202,
            "unit": "ns/op\t 234.52 MB/s\t   42300 B/op\t      10 allocs/op",
            "extra": "14984 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense - ns/op",
            "value": 160202,
            "unit": "ns/op",
            "extra": "14984 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense - MB/s",
            "value": 234.52,
            "unit": "MB/s",
            "extra": "14984 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense - B/op",
            "value": 42300,
            "unit": "B/op",
            "extra": "14984 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "14984 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json",
            "value": 908827,
            "unit": "ns/op\t 271.67 MB/s\t   48346 B/op\t    1001 allocs/op",
            "extra": "2629 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json - ns/op",
            "value": 908827,
            "unit": "ns/op",
            "extra": "2629 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json - MB/s",
            "value": 271.67,
            "unit": "MB/s",
            "extra": "2629 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json - B/op",
            "value": 48346,
            "unit": "B/op",
            "extra": "2629 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json - allocs/op",
            "value": 1001,
            "unit": "allocs/op",
            "extra": "2629 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal",
            "value": 937371,
            "unit": "ns/op\t 263.40 MB/s\t  302282 B/op\t    1002 allocs/op",
            "extra": "2541 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal - ns/op",
            "value": 937371,
            "unit": "ns/op",
            "extra": "2541 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal - MB/s",
            "value": 263.4,
            "unit": "MB/s",
            "extra": "2541 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal - B/op",
            "value": 302282,
            "unit": "B/op",
            "extra": "2541 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal - allocs/op",
            "value": 1002,
            "unit": "allocs/op",
            "extra": "2541 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack",
            "value": 575798,
            "unit": "ns/op\t 322.40 MB/s\t  548384 B/op\t    1015 allocs/op",
            "extra": "4168 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack - ns/op",
            "value": 575798,
            "unit": "ns/op",
            "extra": "4168 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack - MB/s",
            "value": 322.4,
            "unit": "MB/s",
            "extra": "4168 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack - B/op",
            "value": 548384,
            "unit": "B/op",
            "extra": "4168 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack - allocs/op",
            "value": 1015,
            "unit": "allocs/op",
            "extra": "4168 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast",
            "value": 179337,
            "unit": "ns/op\t1035.20 MB/s\t  188695 B/op\t       3 allocs/op",
            "extra": "13382 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast - ns/op",
            "value": 179337,
            "unit": "ns/op",
            "extra": "13382 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast - MB/s",
            "value": 1035.2,
            "unit": "MB/s",
            "extra": "13382 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast - B/op",
            "value": 188695,
            "unit": "B/op",
            "extra": "13382 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "13382 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense",
            "value": 179995,
            "unit": "ns/op\t1031.41 MB/s\t  188832 B/op\t       3 allocs/op",
            "extra": "13314 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense - ns/op",
            "value": 179995,
            "unit": "ns/op",
            "extra": "13314 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense - MB/s",
            "value": 1031.41,
            "unit": "MB/s",
            "extra": "13314 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense - B/op",
            "value": 188832,
            "unit": "B/op",
            "extra": "13314 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "13314 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json",
            "value": 750.1,
            "unit": "ns/op\t  31.99 MB/s\t     248 B/op\t       6 allocs/op",
            "extra": "3189012 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json - ns/op",
            "value": 750.1,
            "unit": "ns/op",
            "extra": "3189012 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json - MB/s",
            "value": 31.99,
            "unit": "MB/s",
            "extra": "3189012 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json - B/op",
            "value": 248,
            "unit": "B/op",
            "extra": "3189012 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "3189012 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack",
            "value": 311.2,
            "unit": "ns/op\t  51.41 MB/s\t      77 B/op\t       3 allocs/op",
            "extra": "7683438 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack - ns/op",
            "value": 311.2,
            "unit": "ns/op",
            "extra": "7683438 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack - MB/s",
            "value": 51.41,
            "unit": "MB/s",
            "extra": "7683438 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack - B/op",
            "value": 77,
            "unit": "B/op",
            "extra": "7683438 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "7683438 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast",
            "value": 157.9,
            "unit": "ns/op\t 139.36 MB/s\t      29 B/op\t       2 allocs/op",
            "extra": "14960704 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast - ns/op",
            "value": 157.9,
            "unit": "ns/op",
            "extra": "14960704 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast - MB/s",
            "value": 139.36,
            "unit": "MB/s",
            "extra": "14960704 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast - B/op",
            "value": 29,
            "unit": "B/op",
            "extra": "14960704 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "14960704 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense",
            "value": 326.3,
            "unit": "ns/op\t  76.61 MB/s\t      72 B/op\t       4 allocs/op",
            "extra": "7365280 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense - ns/op",
            "value": 326.3,
            "unit": "ns/op",
            "extra": "7365280 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense - MB/s",
            "value": 76.61,
            "unit": "MB/s",
            "extra": "7365280 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense - B/op",
            "value": 72,
            "unit": "B/op",
            "extra": "7365280 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "7365280 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json",
            "value": 4585,
            "unit": "ns/op\t  45.80 MB/s\t     448 B/op\t      10 allocs/op",
            "extra": "507242 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json - ns/op",
            "value": 4585,
            "unit": "ns/op",
            "extra": "507242 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json - MB/s",
            "value": 45.8,
            "unit": "MB/s",
            "extra": "507242 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json - B/op",
            "value": 448,
            "unit": "B/op",
            "extra": "507242 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "507242 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack",
            "value": 1587,
            "unit": "ns/op\t  84.41 MB/s\t     272 B/op\t       7 allocs/op",
            "extra": "1511852 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack - ns/op",
            "value": 1587,
            "unit": "ns/op",
            "extra": "1511852 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack - MB/s",
            "value": 84.41,
            "unit": "MB/s",
            "extra": "1511852 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack - B/op",
            "value": 272,
            "unit": "B/op",
            "extra": "1511852 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "1511852 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast",
            "value": 819.6,
            "unit": "ns/op\t 161.06 MB/s\t     224 B/op\t       6 allocs/op",
            "extra": "2914435 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast - ns/op",
            "value": 819.6,
            "unit": "ns/op",
            "extra": "2914435 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast - MB/s",
            "value": 161.06,
            "unit": "MB/s",
            "extra": "2914435 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast - B/op",
            "value": 224,
            "unit": "B/op",
            "extra": "2914435 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "2914435 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense",
            "value": 1409,
            "unit": "ns/op\t  97.94 MB/s\t     624 B/op\t       8 allocs/op",
            "extra": "1709200 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense - ns/op",
            "value": 1409,
            "unit": "ns/op",
            "extra": "1709200 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense - MB/s",
            "value": 97.94,
            "unit": "MB/s",
            "extra": "1709200 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense - B/op",
            "value": 624,
            "unit": "B/op",
            "extra": "1709200 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "1709200 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json",
            "value": 2502,
            "unit": "ns/op\t  41.16 MB/s\t     664 B/op\t      15 allocs/op",
            "extra": "905918 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json - ns/op",
            "value": 2502,
            "unit": "ns/op",
            "extra": "905918 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json - MB/s",
            "value": 41.16,
            "unit": "MB/s",
            "extra": "905918 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json - B/op",
            "value": 664,
            "unit": "B/op",
            "extra": "905918 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "905918 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack",
            "value": 1085,
            "unit": "ns/op\t  70.07 MB/s\t     160 B/op\t       6 allocs/op",
            "extra": "2216091 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack - ns/op",
            "value": 1085,
            "unit": "ns/op",
            "extra": "2216091 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack - MB/s",
            "value": 70.07,
            "unit": "MB/s",
            "extra": "2216091 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack - B/op",
            "value": 160,
            "unit": "B/op",
            "extra": "2216091 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "2216091 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast",
            "value": 398.4,
            "unit": "ns/op\t 215.87 MB/s\t     112 B/op\t       5 allocs/op",
            "extra": "6007663 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast - ns/op",
            "value": 398.4,
            "unit": "ns/op",
            "extra": "6007663 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast - MB/s",
            "value": 215.87,
            "unit": "MB/s",
            "extra": "6007663 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast - B/op",
            "value": 112,
            "unit": "B/op",
            "extra": "6007663 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "6007663 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense",
            "value": 914.6,
            "unit": "ns/op\t 104.96 MB/s\t     296 B/op\t      15 allocs/op",
            "extra": "2594545 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense - ns/op",
            "value": 914.6,
            "unit": "ns/op",
            "extra": "2594545 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense - MB/s",
            "value": 104.96,
            "unit": "MB/s",
            "extra": "2594545 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense - B/op",
            "value": 296,
            "unit": "B/op",
            "extra": "2594545 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "2594545 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json",
            "value": 7004,
            "unit": "ns/op\t  34.12 MB/s\t    1200 B/op\t      29 allocs/op",
            "extra": "335461 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json - ns/op",
            "value": 7004,
            "unit": "ns/op",
            "extra": "335461 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json - MB/s",
            "value": 34.12,
            "unit": "MB/s",
            "extra": "335461 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json - B/op",
            "value": 1200,
            "unit": "B/op",
            "extra": "335461 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json - allocs/op",
            "value": 29,
            "unit": "allocs/op",
            "extra": "335461 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack",
            "value": 3875,
            "unit": "ns/op\t  35.87 MB/s\t     312 B/op\t      18 allocs/op",
            "extra": "598816 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack - ns/op",
            "value": 3875,
            "unit": "ns/op",
            "extra": "598816 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack - MB/s",
            "value": 35.87,
            "unit": "MB/s",
            "extra": "598816 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack - B/op",
            "value": 312,
            "unit": "B/op",
            "extra": "598816 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "598816 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast",
            "value": 1966,
            "unit": "ns/op\t  84.45 MB/s\t     264 B/op\t      17 allocs/op",
            "extra": "1221458 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast - ns/op",
            "value": 1966,
            "unit": "ns/op",
            "extra": "1221458 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast - MB/s",
            "value": 84.45,
            "unit": "MB/s",
            "extra": "1221458 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast - B/op",
            "value": 264,
            "unit": "B/op",
            "extra": "1221458 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "1221458 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense",
            "value": 1936,
            "unit": "ns/op\t  32.55 MB/s\t     304 B/op\t      19 allocs/op",
            "extra": "1239304 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense - ns/op",
            "value": 1936,
            "unit": "ns/op",
            "extra": "1239304 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense - MB/s",
            "value": 32.55,
            "unit": "MB/s",
            "extra": "1239304 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense - B/op",
            "value": 304,
            "unit": "B/op",
            "extra": "1239304 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense - allocs/op",
            "value": 19,
            "unit": "allocs/op",
            "extra": "1239304 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/json",
            "value": 4488902,
            "unit": "ns/op\t  47.43 MB/s\t  638351 B/op\t    5020 allocs/op",
            "extra": "534 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/json - ns/op",
            "value": 4488902,
            "unit": "ns/op",
            "extra": "534 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/json - MB/s",
            "value": 47.43,
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
            "value": 1537928,
            "unit": "ns/op\t  88.19 MB/s\t  409043 B/op\t    5007 allocs/op",
            "extra": "1524 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack - ns/op",
            "value": 1537928,
            "unit": "ns/op",
            "extra": "1524 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack - MB/s",
            "value": 88.19,
            "unit": "MB/s",
            "extra": "1524 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack - B/op",
            "value": 409043,
            "unit": "B/op",
            "extra": "1524 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack - allocs/op",
            "value": 5007,
            "unit": "allocs/op",
            "extra": "1524 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast",
            "value": 731260,
            "unit": "ns/op\t 175.90 MB/s\t  220500 B/op\t    5003 allocs/op",
            "extra": "3250 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast - ns/op",
            "value": 731260,
            "unit": "ns/op",
            "extra": "3250 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast - MB/s",
            "value": 175.9,
            "unit": "MB/s",
            "extra": "3250 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast - B/op",
            "value": 220500,
            "unit": "B/op",
            "extra": "3250 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast - allocs/op",
            "value": 5003,
            "unit": "allocs/op",
            "extra": "3250 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_dense",
            "value": 203931,
            "unit": "ns/op\t 184.23 MB/s\t  318265 B/op\t    5022 allocs/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_dense - ns/op",
            "value": 203931,
            "unit": "ns/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_dense - MB/s",
            "value": 184.23,
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
            "value": 3468285,
            "unit": "ns/op\t  71.19 MB/s\t  442536 B/op\t    7019 allocs/op",
            "extra": "691 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/json - ns/op",
            "value": 3468285,
            "unit": "ns/op",
            "extra": "691 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/json - MB/s",
            "value": 71.19,
            "unit": "MB/s",
            "extra": "691 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/json - B/op",
            "value": 442536,
            "unit": "B/op",
            "extra": "691 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/json - allocs/op",
            "value": 7019,
            "unit": "allocs/op",
            "extra": "691 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack",
            "value": 1078140,
            "unit": "ns/op\t 172.18 MB/s\t  407513 B/op\t    7007 allocs/op",
            "extra": "2206 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack - ns/op",
            "value": 1078140,
            "unit": "ns/op",
            "extra": "2206 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack - MB/s",
            "value": 172.18,
            "unit": "MB/s",
            "extra": "2206 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack - B/op",
            "value": 407513,
            "unit": "B/op",
            "extra": "2206 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack - allocs/op",
            "value": 7007,
            "unit": "allocs/op",
            "extra": "2206 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast",
            "value": 423428,
            "unit": "ns/op\t 438.44 MB/s\t  251713 B/op\t    7002 allocs/op",
            "extra": "5653 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast - ns/op",
            "value": 423428,
            "unit": "ns/op",
            "extra": "5653 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast - MB/s",
            "value": 438.44,
            "unit": "MB/s",
            "extra": "5653 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast - B/op",
            "value": 251713,
            "unit": "B/op",
            "extra": "5653 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast - allocs/op",
            "value": 7002,
            "unit": "allocs/op",
            "extra": "5653 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense",
            "value": 422298,
            "unit": "ns/op\t 439.62 MB/s\t  255169 B/op\t    7005 allocs/op",
            "extra": "5654 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense - ns/op",
            "value": 422298,
            "unit": "ns/op",
            "extra": "5654 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense - MB/s",
            "value": 439.62,
            "unit": "MB/s",
            "extra": "5654 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense - B/op",
            "value": 255169,
            "unit": "B/op",
            "extra": "5654 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense - allocs/op",
            "value": 7005,
            "unit": "allocs/op",
            "extra": "5654 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen",
            "value": 321998,
            "unit": "ns/op\t 576.55 MB/s\t  908027 B/op\t      26 allocs/op",
            "extra": "7394 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen - ns/op",
            "value": 321998,
            "unit": "ns/op",
            "extra": "7394 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen - MB/s",
            "value": 576.55,
            "unit": "MB/s",
            "extra": "7394 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen - B/op",
            "value": 908027,
            "unit": "B/op",
            "extra": "7394 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen - allocs/op",
            "value": 26,
            "unit": "allocs/op",
            "extra": "7394 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen",
            "value": 421467,
            "unit": "ns/op\t 440.48 MB/s\t  251648 B/op\t    7001 allocs/op",
            "extra": "5712 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen - ns/op",
            "value": 421467,
            "unit": "ns/op",
            "extra": "5712 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen - MB/s",
            "value": 440.48,
            "unit": "MB/s",
            "extra": "5712 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen - B/op",
            "value": 251648,
            "unit": "B/op",
            "extra": "5712 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen - allocs/op",
            "value": 7001,
            "unit": "allocs/op",
            "extra": "5712 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json",
            "value": 140863,
            "unit": "ns/op\t 191.14 MB/s\t   27431 B/op\t       2 allocs/op",
            "extra": "17026 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json - ns/op",
            "value": 140863,
            "unit": "ns/op",
            "extra": "17026 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json - MB/s",
            "value": 191.14,
            "unit": "MB/s",
            "extra": "17026 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json - B/op",
            "value": 27431,
            "unit": "B/op",
            "extra": "17026 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "17026 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack",
            "value": 172592,
            "unit": "ns/op\t 220.14 MB/s\t  131235 B/op\t      13 allocs/op",
            "extra": "13898 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack - ns/op",
            "value": 172592,
            "unit": "ns/op",
            "extra": "13898 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack - MB/s",
            "value": 220.14,
            "unit": "MB/s",
            "extra": "13898 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack - B/op",
            "value": 131235,
            "unit": "B/op",
            "extra": "13898 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack - allocs/op",
            "value": 13,
            "unit": "allocs/op",
            "extra": "13898 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf",
            "value": 14850,
            "unit": "ns/op\t 589.85 MB/s\t    9794 B/op\t       3 allocs/op",
            "extra": "160592 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf - ns/op",
            "value": 14850,
            "unit": "ns/op",
            "extra": "160592 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf - MB/s",
            "value": 589.85,
            "unit": "MB/s",
            "extra": "160592 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf - B/op",
            "value": 9794,
            "unit": "B/op",
            "extra": "160592 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "160592 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json",
            "value": 628167,
            "unit": "ns/op\t  42.86 MB/s\t  104576 B/op\t      65 allocs/op",
            "extra": "3760 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json - ns/op",
            "value": 628167,
            "unit": "ns/op",
            "extra": "3760 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json - MB/s",
            "value": 42.86,
            "unit": "MB/s",
            "extra": "3760 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json - B/op",
            "value": 104576,
            "unit": "B/op",
            "extra": "3760 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json - allocs/op",
            "value": 65,
            "unit": "allocs/op",
            "extra": "3760 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack",
            "value": 243575,
            "unit": "ns/op\t 155.98 MB/s\t   68193 B/op\t      29 allocs/op",
            "extra": "9649 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack - ns/op",
            "value": 243575,
            "unit": "ns/op",
            "extra": "9649 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack - MB/s",
            "value": 155.98,
            "unit": "MB/s",
            "extra": "9649 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack - B/op",
            "value": 68193,
            "unit": "B/op",
            "extra": "9649 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack - allocs/op",
            "value": 29,
            "unit": "allocs/op",
            "extra": "9649 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf",
            "value": 12379,
            "unit": "ns/op\t 707.57 MB/s\t   42332 B/op\t      11 allocs/op",
            "extra": "195376 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf - ns/op",
            "value": 12379,
            "unit": "ns/op",
            "extra": "195376 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf - MB/s",
            "value": 707.57,
            "unit": "MB/s",
            "extra": "195376 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf - B/op",
            "value": 42332,
            "unit": "B/op",
            "extra": "195376 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "195376 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json",
            "value": 70108,
            "unit": "ns/op\t 246.96 MB/s\t   18536 B/op\t       2 allocs/op",
            "extra": "34209 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json - ns/op",
            "value": 70108,
            "unit": "ns/op",
            "extra": "34209 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json - MB/s",
            "value": 246.96,
            "unit": "MB/s",
            "extra": "34209 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json - B/op",
            "value": 18536,
            "unit": "B/op",
            "extra": "34209 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "34209 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack",
            "value": 110677,
            "unit": "ns/op\t 250.37 MB/s\t   65625 B/op\t      12 allocs/op",
            "extra": "21772 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack - ns/op",
            "value": 110677,
            "unit": "ns/op",
            "extra": "21772 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack - MB/s",
            "value": 250.37,
            "unit": "MB/s",
            "extra": "21772 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack - B/op",
            "value": 65625,
            "unit": "B/op",
            "extra": "21772 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "21772 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf",
            "value": 12269,
            "unit": "ns/op\t  45.89 MB/s\t     768 B/op\t       3 allocs/op",
            "extra": "193851 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf - ns/op",
            "value": 12269,
            "unit": "ns/op",
            "extra": "193851 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf - MB/s",
            "value": 45.89,
            "unit": "MB/s",
            "extra": "193851 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf - B/op",
            "value": 768,
            "unit": "B/op",
            "extra": "193851 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "193851 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json",
            "value": 385764,
            "unit": "ns/op\t  44.88 MB/s\t   75976 B/op\t      43 allocs/op",
            "extra": "6105 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json - ns/op",
            "value": 385764,
            "unit": "ns/op",
            "extra": "6105 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json - MB/s",
            "value": 44.88,
            "unit": "MB/s",
            "extra": "6105 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json - B/op",
            "value": 75976,
            "unit": "B/op",
            "extra": "6105 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json - allocs/op",
            "value": 43,
            "unit": "allocs/op",
            "extra": "6105 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack",
            "value": 159959,
            "unit": "ns/op\t 173.23 MB/s\t   49543 B/op\t      18 allocs/op",
            "extra": "15004 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack - ns/op",
            "value": 159959,
            "unit": "ns/op",
            "extra": "15004 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack - MB/s",
            "value": 173.23,
            "unit": "MB/s",
            "extra": "15004 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack - B/op",
            "value": 49543,
            "unit": "B/op",
            "extra": "15004 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "15004 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf",
            "value": 9959,
            "unit": "ns/op\t  56.53 MB/s\t   32895 B/op\t       6 allocs/op",
            "extra": "241044 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf - ns/op",
            "value": 9959,
            "unit": "ns/op",
            "extra": "241044 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf - MB/s",
            "value": 56.53,
            "unit": "MB/s",
            "extra": "241044 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf - B/op",
            "value": 32895,
            "unit": "B/op",
            "extra": "241044 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "241044 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json",
            "value": 27943,
            "unit": "ns/op\t 244.32 MB/s\t    6961 B/op\t       2 allocs/op",
            "extra": "85759 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json - ns/op",
            "value": 27943,
            "unit": "ns/op",
            "extra": "85759 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json - MB/s",
            "value": 244.32,
            "unit": "MB/s",
            "extra": "85759 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json - B/op",
            "value": 6961,
            "unit": "B/op",
            "extra": "85759 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "85759 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack",
            "value": 38802,
            "unit": "ns/op\t 238.37 MB/s\t   32804 B/op\t      11 allocs/op",
            "extra": "61675 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack - ns/op",
            "value": 38802,
            "unit": "ns/op",
            "extra": "61675 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack - MB/s",
            "value": 238.37,
            "unit": "MB/s",
            "extra": "61675 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack - B/op",
            "value": 32804,
            "unit": "B/op",
            "extra": "61675 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "61675 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf",
            "value": 11740,
            "unit": "ns/op\t  26.23 MB/s\t     416 B/op\t       3 allocs/op",
            "extra": "202744 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf - ns/op",
            "value": 11740,
            "unit": "ns/op",
            "extra": "202744 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf - MB/s",
            "value": 26.23,
            "unit": "MB/s",
            "extra": "202744 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf - B/op",
            "value": 416,
            "unit": "B/op",
            "extra": "202744 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "202744 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json",
            "value": 148518,
            "unit": "ns/op\t  45.97 MB/s\t   25504 B/op\t      19 allocs/op",
            "extra": "16164 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json - ns/op",
            "value": 148518,
            "unit": "ns/op",
            "extra": "16164 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json - MB/s",
            "value": 45.97,
            "unit": "MB/s",
            "extra": "16164 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json - B/op",
            "value": 25504,
            "unit": "B/op",
            "extra": "16164 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json - allocs/op",
            "value": 19,
            "unit": "allocs/op",
            "extra": "16164 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack",
            "value": 53408,
            "unit": "ns/op\t 173.18 MB/s\t   16570 B/op\t       8 allocs/op",
            "extra": "44684 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack - ns/op",
            "value": 53408,
            "unit": "ns/op",
            "extra": "44684 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack - MB/s",
            "value": 173.18,
            "unit": "MB/s",
            "extra": "44684 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack - B/op",
            "value": 16570,
            "unit": "B/op",
            "extra": "44684 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "44684 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf",
            "value": 5679,
            "unit": "ns/op\t  54.23 MB/s\t   16452 B/op\t       4 allocs/op",
            "extra": "409203 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf - ns/op",
            "value": 5679,
            "unit": "ns/op",
            "extra": "409203 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf - MB/s",
            "value": 54.23,
            "unit": "MB/s",
            "extra": "409203 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf - B/op",
            "value": 16452,
            "unit": "B/op",
            "extra": "409203 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "409203 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json",
            "value": 169478,
            "unit": "ns/op\t 433.42 MB/s\t   73763 B/op\t       2 allocs/op",
            "extra": "14172 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json - ns/op",
            "value": 169478,
            "unit": "ns/op",
            "extra": "14172 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json - MB/s",
            "value": 433.42,
            "unit": "MB/s",
            "extra": "14172 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json - B/op",
            "value": 73763,
            "unit": "B/op",
            "extra": "14172 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "14172 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack",
            "value": 145218,
            "unit": "ns/op\t 411.19 MB/s\t  131100 B/op\t      13 allocs/op",
            "extra": "16526 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack - ns/op",
            "value": 145218,
            "unit": "ns/op",
            "extra": "16526 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack - MB/s",
            "value": 411.19,
            "unit": "MB/s",
            "extra": "16526 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack - B/op",
            "value": 131100,
            "unit": "B/op",
            "extra": "16526 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack - allocs/op",
            "value": 13,
            "unit": "allocs/op",
            "extra": "16526 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf",
            "value": 87030,
            "unit": "ns/op\t 386.16 MB/s\t   41063 B/op\t       3 allocs/op",
            "extra": "27469 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf - ns/op",
            "value": 87030,
            "unit": "ns/op",
            "extra": "27469 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf - MB/s",
            "value": 386.16,
            "unit": "MB/s",
            "extra": "27469 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf - B/op",
            "value": 41063,
            "unit": "B/op",
            "extra": "27469 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "27469 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json",
            "value": 978509,
            "unit": "ns/op\t  75.07 MB/s\t  125256 B/op\t    2016 allocs/op",
            "extra": "2470 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json - ns/op",
            "value": 978509,
            "unit": "ns/op",
            "extra": "2470 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json - MB/s",
            "value": 75.07,
            "unit": "MB/s",
            "extra": "2470 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json - B/op",
            "value": 125256,
            "unit": "B/op",
            "extra": "2470 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json - allocs/op",
            "value": 2016,
            "unit": "allocs/op",
            "extra": "2470 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack",
            "value": 291198,
            "unit": "ns/op\t 205.06 MB/s\t  114785 B/op\t    2007 allocs/op",
            "extra": "7911 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack - ns/op",
            "value": 291198,
            "unit": "ns/op",
            "extra": "7911 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack - MB/s",
            "value": 205.06,
            "unit": "MB/s",
            "extra": "7911 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack - B/op",
            "value": 114785,
            "unit": "B/op",
            "extra": "7911 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack - allocs/op",
            "value": 2007,
            "unit": "allocs/op",
            "extra": "7911 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf",
            "value": 99299,
            "unit": "ns/op\t 338.44 MB/s\t   65203 B/op\t    1012 allocs/op",
            "extra": "24050 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf - ns/op",
            "value": 99299,
            "unit": "ns/op",
            "extra": "24050 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf - MB/s",
            "value": 338.44,
            "unit": "MB/s",
            "extra": "24050 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf - B/op",
            "value": 65203,
            "unit": "B/op",
            "extra": "24050 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf - allocs/op",
            "value": 1012,
            "unit": "allocs/op",
            "extra": "24050 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/json",
            "value": 26475,
            "unit": "ns/op\t    2736 B/op\t       2 allocs/op",
            "extra": "90594 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/json - ns/op",
            "value": 26475,
            "unit": "ns/op",
            "extra": "90594 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/json - B/op",
            "value": 2736,
            "unit": "B/op",
            "extra": "90594 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "90594 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/msgpack",
            "value": 17425,
            "unit": "ns/op\t    8225 B/op\t       9 allocs/op",
            "extra": "138482 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/msgpack - ns/op",
            "value": 17425,
            "unit": "ns/op",
            "extra": "138482 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/msgpack - B/op",
            "value": 8225,
            "unit": "B/op",
            "extra": "138482 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/msgpack - allocs/op",
            "value": 9,
            "unit": "allocs/op",
            "extra": "138482 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/qdf_fast",
            "value": 1929,
            "unit": "ns/op\t    2784 B/op\t       3 allocs/op",
            "extra": "1247607 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/qdf_fast - ns/op",
            "value": 1929,
            "unit": "ns/op",
            "extra": "1247607 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/qdf_fast - B/op",
            "value": 2784,
            "unit": "B/op",
            "extra": "1247607 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1247607 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/json",
            "value": 29764,
            "unit": "ns/op\t    2736 B/op\t       2 allocs/op",
            "extra": "79256 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/json - ns/op",
            "value": 29764,
            "unit": "ns/op",
            "extra": "79256 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/json - B/op",
            "value": 2736,
            "unit": "B/op",
            "extra": "79256 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "79256 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/msgpack",
            "value": 19333,
            "unit": "ns/op\t   16418 B/op\t      10 allocs/op",
            "extra": "123916 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/msgpack - ns/op",
            "value": 19333,
            "unit": "ns/op",
            "extra": "123916 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/msgpack - B/op",
            "value": 16418,
            "unit": "B/op",
            "extra": "123916 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/msgpack - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "123916 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/qdf_fast",
            "value": 2236,
            "unit": "ns/op\t    4961 B/op\t       3 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/qdf_fast - ns/op",
            "value": 2236,
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
            "value": 74522,
            "unit": "ns/op\t    4384 B/op\t      16 allocs/op",
            "extra": "32204 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/json - ns/op",
            "value": 74522,
            "unit": "ns/op",
            "extra": "32204 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/json - B/op",
            "value": 4384,
            "unit": "B/op",
            "extra": "32204 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/json - allocs/op",
            "value": 16,
            "unit": "allocs/op",
            "extra": "32204 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/msgpack",
            "value": 26045,
            "unit": "ns/op\t    4280 B/op\t       8 allocs/op",
            "extra": "92187 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/msgpack - ns/op",
            "value": 26045,
            "unit": "ns/op",
            "extra": "92187 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/msgpack - B/op",
            "value": 4280,
            "unit": "B/op",
            "extra": "92187 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/msgpack - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "92187 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/qdf_fast",
            "value": 3802,
            "unit": "ns/op\t    2112 B/op\t       3 allocs/op",
            "extra": "623260 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/qdf_fast - ns/op",
            "value": 3802,
            "unit": "ns/op",
            "extra": "623260 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/qdf_fast - B/op",
            "value": 2112,
            "unit": "B/op",
            "extra": "623260 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "623260 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/json",
            "value": 152746808,
            "unit": "ns/op\t 243.78 MB/s\t57769314 B/op\t  350217 allocs/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/json - ns/op",
            "value": 152746808,
            "unit": "ns/op",
            "extra": "14 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/json - MB/s",
            "value": 243.78,
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
            "value": 85736840,
            "unit": "ns/op\t 284.23 MB/s\t68709109 B/op\t  100022 allocs/op",
            "extra": "27 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/msgpack - ns/op",
            "value": 85736840,
            "unit": "ns/op",
            "extra": "27 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/msgpack - MB/s",
            "value": 284.23,
            "unit": "MB/s",
            "extra": "27 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/msgpack - B/op",
            "value": 68709109,
            "unit": "B/op",
            "extra": "27 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/msgpack - allocs/op",
            "value": 100022,
            "unit": "allocs/op",
            "extra": "27 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_fast",
            "value": 27297439,
            "unit": "ns/op\t 883.15 MB/s\t29512041 B/op\t      19 allocs/op",
            "extra": "91 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_fast - ns/op",
            "value": 27297439,
            "unit": "ns/op",
            "extra": "91 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_fast - MB/s",
            "value": 883.15,
            "unit": "MB/s",
            "extra": "91 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_fast - B/op",
            "value": 29512041,
            "unit": "B/op",
            "extra": "91 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_fast - allocs/op",
            "value": 19,
            "unit": "allocs/op",
            "extra": "91 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_qpack",
            "value": 24494853,
            "unit": "ns/op\t 959.19 MB/s\t28815726 B/op\t      19 allocs/op",
            "extra": "100 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_qpack - ns/op",
            "value": 24494853,
            "unit": "ns/op",
            "extra": "100 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_qpack - MB/s",
            "value": 959.19,
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
            "value": 27053828,
            "unit": "ns/op\t 668.51 MB/s\t24114446 B/op\t      74 allocs/op",
            "extra": "86 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_dense - ns/op",
            "value": 27053828,
            "unit": "ns/op",
            "extra": "86 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_dense - MB/s",
            "value": 668.51,
            "unit": "MB/s",
            "extra": "86 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_dense - B/op",
            "value": 24114446,
            "unit": "B/op",
            "extra": "86 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_dense - allocs/op",
            "value": 74,
            "unit": "allocs/op",
            "extra": "86 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/json",
            "value": 619693591,
            "unit": "ns/op\t  60.10 MB/s\t119804004 B/op\t 1559637 allocs/op",
            "extra": "4 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/json - ns/op",
            "value": 619693591,
            "unit": "ns/op",
            "extra": "4 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/json - MB/s",
            "value": 60.1,
            "unit": "MB/s",
            "extra": "4 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/json - B/op",
            "value": 119804004,
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
            "value": 185518433,
            "unit": "ns/op\t 131.38 MB/s\t74391021 B/op\t 1425125 allocs/op",
            "extra": "12 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/msgpack - ns/op",
            "value": 185518433,
            "unit": "ns/op",
            "extra": "12 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/msgpack - MB/s",
            "value": 131.38,
            "unit": "MB/s",
            "extra": "12 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/msgpack - B/op",
            "value": 74391021,
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
            "value": 72921482,
            "unit": "ns/op\t 330.65 MB/s\t48380087 B/op\t  875099 allocs/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_fast - ns/op",
            "value": 72921482,
            "unit": "ns/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_fast - MB/s",
            "value": 330.65,
            "unit": "MB/s",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_fast - B/op",
            "value": 48380087,
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
            "value": 68052574,
            "unit": "ns/op\t 345.29 MB/s\t48380635 B/op\t  875099 allocs/op",
            "extra": "33 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_qpack - ns/op",
            "value": 68052574,
            "unit": "ns/op",
            "extra": "33 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_qpack - MB/s",
            "value": 345.29,
            "unit": "MB/s",
            "extra": "33 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_qpack - B/op",
            "value": 48380635,
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
            "value": 62616735,
            "unit": "ns/op\t 288.86 MB/s\t50890971 B/op\t  790952 allocs/op",
            "extra": "39 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense - ns/op",
            "value": 62616735,
            "unit": "ns/op",
            "extra": "39 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense - MB/s",
            "value": 288.86,
            "unit": "MB/s",
            "extra": "39 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense - B/op",
            "value": 50890971,
            "unit": "B/op",
            "extra": "39 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense - allocs/op",
            "value": 790952,
            "unit": "allocs/op",
            "extra": "39 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/json",
            "value": 7993,
            "unit": "ns/op\t    3408 B/op\t      84 allocs/op",
            "extra": "284692 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/json - ns/op",
            "value": 7993,
            "unit": "ns/op",
            "extra": "284692 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/json - B/op",
            "value": 3408,
            "unit": "B/op",
            "extra": "284692 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/json - allocs/op",
            "value": 84,
            "unit": "allocs/op",
            "extra": "284692 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/msgpack",
            "value": 4300,
            "unit": "ns/op\t    1536 B/op\t      46 allocs/op",
            "extra": "545382 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/msgpack - ns/op",
            "value": 4300,
            "unit": "ns/op",
            "extra": "545382 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/msgpack - B/op",
            "value": 1536,
            "unit": "B/op",
            "extra": "545382 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/msgpack - allocs/op",
            "value": 46,
            "unit": "allocs/op",
            "extra": "545382 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_fast",
            "value": 1469,
            "unit": "ns/op\t     416 B/op\t       3 allocs/op",
            "extra": "1633695 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_fast - ns/op",
            "value": 1469,
            "unit": "ns/op",
            "extra": "1633695 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_fast - B/op",
            "value": 416,
            "unit": "B/op",
            "extra": "1633695 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1633695 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_dense",
            "value": 1725,
            "unit": "ns/op\t     416 B/op\t       3 allocs/op",
            "extra": "1389993 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_dense - ns/op",
            "value": 1725,
            "unit": "ns/op",
            "extra": "1389993 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_dense - B/op",
            "value": 416,
            "unit": "B/op",
            "extra": "1389993 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1389993 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/json",
            "value": 16952,
            "unit": "ns/op\t    4912 B/op\t     124 allocs/op",
            "extra": "140517 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/json - ns/op",
            "value": 16952,
            "unit": "ns/op",
            "extra": "140517 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/json - B/op",
            "value": 4912,
            "unit": "B/op",
            "extra": "140517 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/json - allocs/op",
            "value": 124,
            "unit": "allocs/op",
            "extra": "140517 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/msgpack",
            "value": 7207,
            "unit": "ns/op\t    3088 B/op\t     112 allocs/op",
            "extra": "331873 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/msgpack - ns/op",
            "value": 7207,
            "unit": "ns/op",
            "extra": "331873 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/msgpack - B/op",
            "value": 3088,
            "unit": "B/op",
            "extra": "331873 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/msgpack - allocs/op",
            "value": 112,
            "unit": "allocs/op",
            "extra": "331873 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/qdf_fast",
            "value": 2945,
            "unit": "ns/op\t    2355 B/op\t      32 allocs/op",
            "extra": "832038 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/qdf_fast - ns/op",
            "value": 2945,
            "unit": "ns/op",
            "extra": "832038 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/qdf_fast - B/op",
            "value": 2355,
            "unit": "B/op",
            "extra": "832038 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/qdf_fast - allocs/op",
            "value": 32,
            "unit": "allocs/op",
            "extra": "832038 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/json",
            "value": 9199,
            "unit": "ns/op\t    2820 B/op\t      71 allocs/op",
            "extra": "257214 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/json - ns/op",
            "value": 9199,
            "unit": "ns/op",
            "extra": "257214 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/json - B/op",
            "value": 2820,
            "unit": "B/op",
            "extra": "257214 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/json - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "257214 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/msgpack",
            "value": 2345,
            "unit": "ns/op\t    1487 B/op\t      46 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/msgpack - ns/op",
            "value": 2345,
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
            "value": 1672,
            "unit": "ns/op\t    1403 B/op\t      26 allocs/op",
            "extra": "1432855 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/qdf_fast - ns/op",
            "value": 1672,
            "unit": "ns/op",
            "extra": "1432855 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/qdf_fast - B/op",
            "value": 1403,
            "unit": "B/op",
            "extra": "1432855 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/qdf_fast - allocs/op",
            "value": 26,
            "unit": "allocs/op",
            "extra": "1432855 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/json",
            "value": 0.3455,
            "unit": "ns/op\t    442537 B/decode\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/json - ns/op",
            "value": 0.3455,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/json - B/decode",
            "value": 442537,
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
            "value": 0.1107,
            "unit": "ns/op\t    407515 B/decode\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/msgpack - ns/op",
            "value": 0.1107,
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
            "value": 0.04384,
            "unit": "ns/op\t    251762 B/decode\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/qdf_fast - ns/op",
            "value": 0.04384,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/qdf_fast - B/decode",
            "value": 251762,
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
            "value": 3433,
            "unit": "ns/op\t     790 B/op\t      37 allocs/op",
            "extra": "668991 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/json - ns/op",
            "value": 3433,
            "unit": "ns/op",
            "extra": "668991 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/json - B/op",
            "value": 790,
            "unit": "B/op",
            "extra": "668991 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/json - allocs/op",
            "value": 37,
            "unit": "allocs/op",
            "extra": "668991 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/qdf_fast",
            "value": 594.9,
            "unit": "ns/op\t     346 B/op\t       3 allocs/op",
            "extra": "4101130 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/qdf_fast - ns/op",
            "value": 594.9,
            "unit": "ns/op",
            "extra": "4101130 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/qdf_fast - B/op",
            "value": 346,
            "unit": "B/op",
            "extra": "4101130 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "4101130 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json",
            "value": 625.2,
            "unit": "ns/op\t 155.16 MB/s\t     240 B/op\t       3 allocs/op",
            "extra": "3836155 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json - ns/op",
            "value": 625.2,
            "unit": "ns/op",
            "extra": "3836155 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json - MB/s",
            "value": 155.16,
            "unit": "MB/s",
            "extra": "3836155 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json - B/op",
            "value": 240,
            "unit": "B/op",
            "extra": "3836155 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3836155 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack",
            "value": 385.2,
            "unit": "ns/op\t 163.57 MB/s\t     192 B/op\t       3 allocs/op",
            "extra": "6221536 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack - ns/op",
            "value": 385.2,
            "unit": "ns/op",
            "extra": "6221536 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack - MB/s",
            "value": 163.57,
            "unit": "MB/s",
            "extra": "6221536 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack - B/op",
            "value": 192,
            "unit": "B/op",
            "extra": "6221536 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "6221536 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf",
            "value": 393.3,
            "unit": "ns/op\t 183.04 MB/s\t     240 B/op\t       3 allocs/op",
            "extra": "6092707 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf - ns/op",
            "value": 393.3,
            "unit": "ns/op",
            "extra": "6092707 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf - MB/s",
            "value": 183.04,
            "unit": "MB/s",
            "extra": "6092707 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf - B/op",
            "value": 240,
            "unit": "B/op",
            "extra": "6092707 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "6092707 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json",
            "value": 1672,
            "unit": "ns/op\t  58.03 MB/s\t     328 B/op\t       7 allocs/op",
            "extra": "1432344 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json - ns/op",
            "value": 1672,
            "unit": "ns/op",
            "extra": "1432344 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json - MB/s",
            "value": 58.03,
            "unit": "MB/s",
            "extra": "1432344 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json - B/op",
            "value": 328,
            "unit": "B/op",
            "extra": "1432344 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "1432344 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack",
            "value": 615.9,
            "unit": "ns/op\t 102.28 MB/s\t     160 B/op\t       4 allocs/op",
            "extra": "3877304 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack - ns/op",
            "value": 615.9,
            "unit": "ns/op",
            "extra": "3877304 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack - MB/s",
            "value": 102.28,
            "unit": "MB/s",
            "extra": "3877304 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack - B/op",
            "value": 160,
            "unit": "B/op",
            "extra": "3877304 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "3877304 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf",
            "value": 271.6,
            "unit": "ns/op\t 265.11 MB/s\t     112 B/op\t       3 allocs/op",
            "extra": "8845063 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf - ns/op",
            "value": 271.6,
            "unit": "ns/op",
            "extra": "8845063 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf - MB/s",
            "value": 265.11,
            "unit": "MB/s",
            "extra": "8845063 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf - B/op",
            "value": 112,
            "unit": "B/op",
            "extra": "8845063 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "8845063 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json",
            "value": 356143,
            "unit": "ns/op\t 401.19 MB/s\t  147907 B/op\t       2 allocs/op",
            "extra": "6525 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json - ns/op",
            "value": 356143,
            "unit": "ns/op",
            "extra": "6525 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json - MB/s",
            "value": 401.19,
            "unit": "MB/s",
            "extra": "6525 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json - B/op",
            "value": 147907,
            "unit": "B/op",
            "extra": "6525 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "6525 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack",
            "value": 460536,
            "unit": "ns/op\t 242.41 MB/s\t  286202 B/op\t    1014 allocs/op",
            "extra": "5162 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack - ns/op",
            "value": 460536,
            "unit": "ns/op",
            "extra": "5162 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack - MB/s",
            "value": 242.41,
            "unit": "MB/s",
            "extra": "5162 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack - B/op",
            "value": 286202,
            "unit": "B/op",
            "extra": "5162 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack - allocs/op",
            "value": 1014,
            "unit": "allocs/op",
            "extra": "5162 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf",
            "value": 256619,
            "unit": "ns/op\t 158.07 MB/s\t   41240 B/op\t       3 allocs/op",
            "extra": "9241 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf - ns/op",
            "value": 256619,
            "unit": "ns/op",
            "extra": "9241 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf - MB/s",
            "value": 158.07,
            "unit": "MB/s",
            "extra": "9241 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf - B/op",
            "value": 41240,
            "unit": "B/op",
            "extra": "9241 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "9241 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json",
            "value": 2655643,
            "unit": "ns/op\t  53.80 MB/s\t  503578 B/op\t    9019 allocs/op",
            "extra": "918 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json - ns/op",
            "value": 2655643,
            "unit": "ns/op",
            "extra": "918 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json - MB/s",
            "value": 53.8,
            "unit": "MB/s",
            "extra": "918 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json - B/op",
            "value": 503578,
            "unit": "B/op",
            "extra": "918 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json - allocs/op",
            "value": 9019,
            "unit": "allocs/op",
            "extra": "918 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack",
            "value": 910947,
            "unit": "ns/op\t 122.55 MB/s\t  323890 B/op\t    8007 allocs/op",
            "extra": "2613 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack - ns/op",
            "value": 910947,
            "unit": "ns/op",
            "extra": "2613 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack - MB/s",
            "value": 122.55,
            "unit": "MB/s",
            "extra": "2613 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack - B/op",
            "value": 323890,
            "unit": "B/op",
            "extra": "2613 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack - allocs/op",
            "value": 8007,
            "unit": "allocs/op",
            "extra": "2613 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf",
            "value": 324614,
            "unit": "ns/op\t 124.96 MB/s\t  169218 B/op\t    3468 allocs/op",
            "extra": "7363 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf - ns/op",
            "value": 324614,
            "unit": "ns/op",
            "extra": "7363 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf - MB/s",
            "value": 124.96,
            "unit": "MB/s",
            "extra": "7363 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf - B/op",
            "value": 169218,
            "unit": "B/op",
            "extra": "7363 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf - allocs/op",
            "value": 3468,
            "unit": "allocs/op",
            "extra": "7363 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf",
            "value": 250490,
            "unit": "ns/op\t 161.93 MB/s\t      91 B/op\t       2 allocs/op",
            "extra": "9500 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf - ns/op",
            "value": 250490,
            "unit": "ns/op",
            "extra": "9500 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf - MB/s",
            "value": 161.93,
            "unit": "MB/s",
            "extra": "9500 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf - B/op",
            "value": 91,
            "unit": "B/op",
            "extra": "9500 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "9500 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json",
            "value": 144049,
            "unit": "ns/op\t 258.65 MB/s\t   41117 B/op\t       2 allocs/op",
            "extra": "16648 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json - ns/op",
            "value": 144049,
            "unit": "ns/op",
            "extra": "16648 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json - MB/s",
            "value": 258.65,
            "unit": "MB/s",
            "extra": "16648 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json - B/op",
            "value": 41117,
            "unit": "B/op",
            "extra": "16648 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "16648 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack",
            "value": 102869,
            "unit": "ns/op\t 189.68 MB/s\t   65626 B/op\t      12 allocs/op",
            "extra": "23031 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack - ns/op",
            "value": 102869,
            "unit": "ns/op",
            "extra": "23031 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack - MB/s",
            "value": 189.68,
            "unit": "MB/s",
            "extra": "23031 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack - B/op",
            "value": 65626,
            "unit": "B/op",
            "extra": "23031 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "23031 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf",
            "value": 4187,
            "unit": "ns/op\t2003.96 MB/s\t    9668 B/op\t       3 allocs/op",
            "extra": "570958 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf - ns/op",
            "value": 4187,
            "unit": "ns/op",
            "extra": "570958 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf - MB/s",
            "value": 2003.96,
            "unit": "MB/s",
            "extra": "570958 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf - B/op",
            "value": 9668,
            "unit": "B/op",
            "extra": "570958 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "570958 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json",
            "value": 551781,
            "unit": "ns/op\t  67.52 MB/s\t   54080 B/op\t      40 allocs/op",
            "extra": "4329 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json - ns/op",
            "value": 551781,
            "unit": "ns/op",
            "extra": "4329 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json - MB/s",
            "value": 67.52,
            "unit": "MB/s",
            "extra": "4329 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json - B/op",
            "value": 54080,
            "unit": "B/op",
            "extra": "4329 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json - allocs/op",
            "value": 40,
            "unit": "allocs/op",
            "extra": "4329 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack",
            "value": 142820,
            "unit": "ns/op\t 136.62 MB/s\t   35197 B/op\t      18 allocs/op",
            "extra": "16824 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack - ns/op",
            "value": 142820,
            "unit": "ns/op",
            "extra": "16824 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack - MB/s",
            "value": 136.62,
            "unit": "MB/s",
            "extra": "16824 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack - B/op",
            "value": 35197,
            "unit": "B/op",
            "extra": "16824 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "16824 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf",
            "value": 4791,
            "unit": "ns/op\t1751.41 MB/s\t   17524 B/op\t       5 allocs/op",
            "extra": "510728 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf - ns/op",
            "value": 4791,
            "unit": "ns/op",
            "extra": "510728 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf - MB/s",
            "value": 1751.41,
            "unit": "MB/s",
            "extra": "510728 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf - B/op",
            "value": 17524,
            "unit": "B/op",
            "extra": "510728 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "510728 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json",
            "value": 119508,
            "unit": "ns/op\t 242.86 MB/s\t   32888 B/op\t       2 allocs/op",
            "extra": "20106 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json - ns/op",
            "value": 119508,
            "unit": "ns/op",
            "extra": "20106 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json - MB/s",
            "value": 242.86,
            "unit": "MB/s",
            "extra": "20106 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json - B/op",
            "value": 32888,
            "unit": "B/op",
            "extra": "20106 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "20106 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack",
            "value": 102503,
            "unit": "ns/op\t 190.42 MB/s\t   65626 B/op\t      12 allocs/op",
            "extra": "23425 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack - ns/op",
            "value": 102503,
            "unit": "ns/op",
            "extra": "23425 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack - MB/s",
            "value": 190.42,
            "unit": "MB/s",
            "extra": "23425 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack - B/op",
            "value": 65626,
            "unit": "B/op",
            "extra": "23425 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "23425 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf",
            "value": 4252,
            "unit": "ns/op\t1975.28 MB/s\t    9668 B/op\t       3 allocs/op",
            "extra": "559501 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf - ns/op",
            "value": 4252,
            "unit": "ns/op",
            "extra": "559501 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf - MB/s",
            "value": 1975.28,
            "unit": "MB/s",
            "extra": "559501 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf - B/op",
            "value": 9668,
            "unit": "B/op",
            "extra": "559501 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "559501 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json",
            "value": 469913,
            "unit": "ns/op\t  61.76 MB/s\t   54088 B/op\t      40 allocs/op",
            "extra": "5042 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json - ns/op",
            "value": 469913,
            "unit": "ns/op",
            "extra": "5042 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json - MB/s",
            "value": 61.76,
            "unit": "MB/s",
            "extra": "5042 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json - B/op",
            "value": 54088,
            "unit": "B/op",
            "extra": "5042 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json - allocs/op",
            "value": 40,
            "unit": "allocs/op",
            "extra": "5042 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack",
            "value": 142674,
            "unit": "ns/op\t 136.81 MB/s\t   35205 B/op\t      18 allocs/op",
            "extra": "16920 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack - ns/op",
            "value": 142674,
            "unit": "ns/op",
            "extra": "16920 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack - MB/s",
            "value": 136.81,
            "unit": "MB/s",
            "extra": "16920 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack - B/op",
            "value": 35205,
            "unit": "B/op",
            "extra": "16920 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "16920 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf",
            "value": 4732,
            "unit": "ns/op\t1774.69 MB/s\t   17533 B/op\t       5 allocs/op",
            "extra": "505864 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf - ns/op",
            "value": 4732,
            "unit": "ns/op",
            "extra": "505864 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf - MB/s",
            "value": 1774.69,
            "unit": "MB/s",
            "extra": "505864 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf - B/op",
            "value": 17533,
            "unit": "B/op",
            "extra": "505864 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "505864 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json",
            "value": 119927,
            "unit": "ns/op\t 242.01 MB/s\t   32885 B/op\t       2 allocs/op",
            "extra": "20000 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json - ns/op",
            "value": 119927,
            "unit": "ns/op",
            "extra": "20000 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json - MB/s",
            "value": 242.01,
            "unit": "MB/s",
            "extra": "20000 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json - B/op",
            "value": 32885,
            "unit": "B/op",
            "extra": "20000 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "20000 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack",
            "value": 102607,
            "unit": "ns/op\t 190.23 MB/s\t   65626 B/op\t      12 allocs/op",
            "extra": "23409 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack - ns/op",
            "value": 102607,
            "unit": "ns/op",
            "extra": "23409 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack - MB/s",
            "value": 190.23,
            "unit": "MB/s",
            "extra": "23409 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack - B/op",
            "value": 65626,
            "unit": "B/op",
            "extra": "23409 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "23409 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf",
            "value": 49090,
            "unit": "ns/op\t  46.99 MB/s\t   11323 B/op\t      14 allocs/op",
            "extra": "48998 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf - ns/op",
            "value": 49090,
            "unit": "ns/op",
            "extra": "48998 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf - MB/s",
            "value": 46.99,
            "unit": "MB/s",
            "extra": "48998 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf - B/op",
            "value": 11323,
            "unit": "B/op",
            "extra": "48998 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf - allocs/op",
            "value": 14,
            "unit": "allocs/op",
            "extra": "48998 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json",
            "value": 468793,
            "unit": "ns/op\t  61.91 MB/s\t   54088 B/op\t      40 allocs/op",
            "extra": "5128 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json - ns/op",
            "value": 468793,
            "unit": "ns/op",
            "extra": "5128 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json - MB/s",
            "value": 61.91,
            "unit": "MB/s",
            "extra": "5128 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json - B/op",
            "value": 54088,
            "unit": "B/op",
            "extra": "5128 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json - allocs/op",
            "value": 40,
            "unit": "allocs/op",
            "extra": "5128 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack",
            "value": 142082,
            "unit": "ns/op\t 137.38 MB/s\t   35205 B/op\t      18 allocs/op",
            "extra": "16914 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack - ns/op",
            "value": 142082,
            "unit": "ns/op",
            "extra": "16914 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack - MB/s",
            "value": 137.38,
            "unit": "MB/s",
            "extra": "16914 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack - B/op",
            "value": 35205,
            "unit": "B/op",
            "extra": "16914 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "16914 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf",
            "value": 43679,
            "unit": "ns/op\t  52.82 MB/s\t   17612 B/op\t       7 allocs/op",
            "extra": "54670 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf - ns/op",
            "value": 43679,
            "unit": "ns/op",
            "extra": "54670 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf - MB/s",
            "value": 52.82,
            "unit": "MB/s",
            "extra": "54670 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf - B/op",
            "value": 17612,
            "unit": "B/op",
            "extra": "54670 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "54670 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json",
            "value": 22968,
            "unit": "ns/op\t 180.08 MB/s\t    4913 B/op\t       2 allocs/op",
            "extra": "105231 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json - ns/op",
            "value": 22968,
            "unit": "ns/op",
            "extra": "105231 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json - MB/s",
            "value": 180.08,
            "unit": "MB/s",
            "extra": "105231 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json - B/op",
            "value": 4913,
            "unit": "B/op",
            "extra": "105231 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "105231 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack",
            "value": 39973,
            "unit": "ns/op\t 231.43 MB/s\t   32805 B/op\t      11 allocs/op",
            "extra": "60156 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack - ns/op",
            "value": 39973,
            "unit": "ns/op",
            "extra": "60156 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack - MB/s",
            "value": 231.43,
            "unit": "MB/s",
            "extra": "60156 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack - B/op",
            "value": 32805,
            "unit": "B/op",
            "extra": "60156 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "60156 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf",
            "value": 4947,
            "unit": "ns/op\t  20.01 MB/s\t     208 B/op\t       3 allocs/op",
            "extra": "491268 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf - ns/op",
            "value": 4947,
            "unit": "ns/op",
            "extra": "491268 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf - MB/s",
            "value": 20.01,
            "unit": "MB/s",
            "extra": "491268 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf - B/op",
            "value": 208,
            "unit": "B/op",
            "extra": "491268 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "491268 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json",
            "value": 121324,
            "unit": "ns/op\t  34.09 MB/s\t   25504 B/op\t      19 allocs/op",
            "extra": "19717 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json - ns/op",
            "value": 121324,
            "unit": "ns/op",
            "extra": "19717 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json - MB/s",
            "value": 34.09,
            "unit": "MB/s",
            "extra": "19717 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json - B/op",
            "value": 25504,
            "unit": "B/op",
            "extra": "19717 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json - allocs/op",
            "value": 19,
            "unit": "allocs/op",
            "extra": "19717 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack",
            "value": 56618,
            "unit": "ns/op\t 163.39 MB/s\t   16570 B/op\t       8 allocs/op",
            "extra": "42772 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack - ns/op",
            "value": 56618,
            "unit": "ns/op",
            "extra": "42772 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack - MB/s",
            "value": 163.39,
            "unit": "MB/s",
            "extra": "42772 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack - B/op",
            "value": 16570,
            "unit": "B/op",
            "extra": "42772 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "42772 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf",
            "value": 2472,
            "unit": "ns/op\t  40.05 MB/s\t    8258 B/op\t       3 allocs/op",
            "extra": "955041 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf - ns/op",
            "value": 2472,
            "unit": "ns/op",
            "extra": "955041 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf - MB/s",
            "value": 40.05,
            "unit": "MB/s",
            "extra": "955041 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf - B/op",
            "value": 8258,
            "unit": "B/op",
            "extra": "955041 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "955041 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json",
            "value": 57362,
            "unit": "ns/op\t 146.18 MB/s\t    9523 B/op\t       2 allocs/op",
            "extra": "41907 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json - ns/op",
            "value": 57362,
            "unit": "ns/op",
            "extra": "41907 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json - MB/s",
            "value": 146.18,
            "unit": "MB/s",
            "extra": "41907 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json - B/op",
            "value": 9523,
            "unit": "B/op",
            "extra": "41907 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "41907 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack",
            "value": 25308,
            "unit": "ns/op\t 152.68 MB/s\t    8225 B/op\t       9 allocs/op",
            "extra": "93780 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack - ns/op",
            "value": 25308,
            "unit": "ns/op",
            "extra": "93780 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack - MB/s",
            "value": 152.68,
            "unit": "MB/s",
            "extra": "93780 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack - B/op",
            "value": 8225,
            "unit": "B/op",
            "extra": "93780 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack - allocs/op",
            "value": 9,
            "unit": "allocs/op",
            "extra": "93780 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf",
            "value": 944.8,
            "unit": "ns/op\t3284.37 MB/s\t    3297 B/op\t       3 allocs/op",
            "extra": "2538556 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf - ns/op",
            "value": 944.8,
            "unit": "ns/op",
            "extra": "2538556 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf - MB/s",
            "value": 3284.37,
            "unit": "MB/s",
            "extra": "2538556 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf - B/op",
            "value": 3297,
            "unit": "B/op",
            "extra": "2538556 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "2538556 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json",
            "value": 162587,
            "unit": "ns/op\t  51.57 MB/s\t    7832 B/op\t      17 allocs/op",
            "extra": "14752 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json - ns/op",
            "value": 162587,
            "unit": "ns/op",
            "extra": "14752 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json - MB/s",
            "value": 51.57,
            "unit": "MB/s",
            "extra": "14752 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json - B/op",
            "value": 7832,
            "unit": "B/op",
            "extra": "14752 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "14752 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack",
            "value": 38827,
            "unit": "ns/op\t  99.52 MB/s\t    6320 B/op\t       8 allocs/op",
            "extra": "61129 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack - ns/op",
            "value": 38827,
            "unit": "ns/op",
            "extra": "61129 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack - MB/s",
            "value": 99.52,
            "unit": "MB/s",
            "extra": "61129 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack - B/op",
            "value": 6320,
            "unit": "B/op",
            "extra": "61129 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "61129 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf",
            "value": 759.5,
            "unit": "ns/op\t4085.46 MB/s\t    3129 B/op\t       3 allocs/op",
            "extra": "3137906 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf - ns/op",
            "value": 759.5,
            "unit": "ns/op",
            "extra": "3137906 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf - MB/s",
            "value": 4085.46,
            "unit": "MB/s",
            "extra": "3137906 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf - B/op",
            "value": 3129,
            "unit": "B/op",
            "extra": "3137906 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3137906 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json",
            "value": 1962,
            "unit": "ns/op\t 127.43 MB/s\t     936 B/op\t      22 allocs/op",
            "extra": "1225500 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json - ns/op",
            "value": 1962,
            "unit": "ns/op",
            "extra": "1225500 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json - MB/s",
            "value": 127.43,
            "unit": "MB/s",
            "extra": "1225500 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json - B/op",
            "value": 936,
            "unit": "B/op",
            "extra": "1225500 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json - allocs/op",
            "value": 22,
            "unit": "allocs/op",
            "extra": "1225500 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack",
            "value": 1587,
            "unit": "ns/op\t 124.13 MB/s\t     680 B/op\t      15 allocs/op",
            "extra": "1508192 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack - ns/op",
            "value": 1587,
            "unit": "ns/op",
            "extra": "1508192 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack - MB/s",
            "value": 124.13,
            "unit": "MB/s",
            "extra": "1508192 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack - B/op",
            "value": 680,
            "unit": "B/op",
            "extra": "1508192 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "1508192 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf",
            "value": 1271,
            "unit": "ns/op\t 177.00 MB/s\t     368 B/op\t       3 allocs/op",
            "extra": "1891856 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf - ns/op",
            "value": 1271,
            "unit": "ns/op",
            "extra": "1891856 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf - MB/s",
            "value": 177,
            "unit": "MB/s",
            "extra": "1891856 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf - B/op",
            "value": 368,
            "unit": "B/op",
            "extra": "1891856 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1891856 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json",
            "value": 5761,
            "unit": "ns/op\t  43.40 MB/s\t    1352 B/op\t      41 allocs/op",
            "extra": "419278 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json - ns/op",
            "value": 5761,
            "unit": "ns/op",
            "extra": "419278 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json - MB/s",
            "value": 43.4,
            "unit": "MB/s",
            "extra": "419278 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json - B/op",
            "value": 1352,
            "unit": "B/op",
            "extra": "419278 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json - allocs/op",
            "value": 41,
            "unit": "allocs/op",
            "extra": "419278 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack",
            "value": 2559,
            "unit": "ns/op\t  77.00 MB/s\t    1064 B/op\t      34 allocs/op",
            "extra": "903188 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack - ns/op",
            "value": 2559,
            "unit": "ns/op",
            "extra": "903188 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack - MB/s",
            "value": 77,
            "unit": "MB/s",
            "extra": "903188 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack - B/op",
            "value": 1064,
            "unit": "B/op",
            "extra": "903188 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack - allocs/op",
            "value": 34,
            "unit": "allocs/op",
            "extra": "903188 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf",
            "value": 1510,
            "unit": "ns/op\t 148.97 MB/s\t     889 B/op\t      16 allocs/op",
            "extra": "1582222 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf - ns/op",
            "value": 1510,
            "unit": "ns/op",
            "extra": "1582222 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf - MB/s",
            "value": 148.97,
            "unit": "MB/s",
            "extra": "1582222 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf - B/op",
            "value": 889,
            "unit": "B/op",
            "extra": "1582222 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf - allocs/op",
            "value": 16,
            "unit": "allocs/op",
            "extra": "1582222 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json",
            "value": 1775693,
            "unit": "ns/op\t 402.54 MB/s\t  730657 B/op\t       3 allocs/op",
            "extra": "1324 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json - ns/op",
            "value": 1775693,
            "unit": "ns/op",
            "extra": "1324 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json - MB/s",
            "value": 402.54,
            "unit": "MB/s",
            "extra": "1324 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json - B/op",
            "value": 730657,
            "unit": "B/op",
            "extra": "1324 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1324 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack",
            "value": 2421563,
            "unit": "ns/op\t 230.64 MB/s\t 2217447 B/op\t    5018 allocs/op",
            "extra": "985 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack - ns/op",
            "value": 2421563,
            "unit": "ns/op",
            "extra": "985 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack - MB/s",
            "value": 230.64,
            "unit": "MB/s",
            "extra": "985 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack - B/op",
            "value": 2217447,
            "unit": "B/op",
            "extra": "985 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack - allocs/op",
            "value": 5018,
            "unit": "allocs/op",
            "extra": "985 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf",
            "value": 1828116,
            "unit": "ns/op\t 105.16 MB/s\t 1518378 B/op\t      51 allocs/op",
            "extra": "1339 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf - ns/op",
            "value": 1828116,
            "unit": "ns/op",
            "extra": "1339 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf - MB/s",
            "value": 105.16,
            "unit": "MB/s",
            "extra": "1339 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf - B/op",
            "value": 1518378,
            "unit": "B/op",
            "extra": "1339 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "1339 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json",
            "value": 13199805,
            "unit": "ns/op\t  54.15 MB/s\t 3074427 B/op\t   45025 allocs/op",
            "extra": "181 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json - ns/op",
            "value": 13199805,
            "unit": "ns/op",
            "extra": "181 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json - MB/s",
            "value": 54.15,
            "unit": "MB/s",
            "extra": "181 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json - B/op",
            "value": 3074427,
            "unit": "B/op",
            "extra": "181 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json - allocs/op",
            "value": 45025,
            "unit": "allocs/op",
            "extra": "181 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack",
            "value": 4832998,
            "unit": "ns/op\t 115.56 MB/s\t 1602198 B/op\t   40008 allocs/op",
            "extra": "496 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack - ns/op",
            "value": 4832998,
            "unit": "ns/op",
            "extra": "496 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack - MB/s",
            "value": 115.56,
            "unit": "MB/s",
            "extra": "496 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack - B/op",
            "value": 1602198,
            "unit": "B/op",
            "extra": "496 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack - allocs/op",
            "value": 40008,
            "unit": "allocs/op",
            "extra": "496 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf",
            "value": 2386949,
            "unit": "ns/op\t  80.54 MB/s\t 1823005 B/op\t   16297 allocs/op",
            "extra": "1003 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf - ns/op",
            "value": 2386949,
            "unit": "ns/op",
            "extra": "1003 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf - MB/s",
            "value": 80.54,
            "unit": "MB/s",
            "extra": "1003 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf - B/op",
            "value": 1823005,
            "unit": "B/op",
            "extra": "1003 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf - allocs/op",
            "value": 16297,
            "unit": "allocs/op",
            "extra": "1003 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/json",
            "value": 45279,
            "unit": "ns/op\t   10982 B/op\t       2 allocs/op",
            "extra": "52886 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/json - ns/op",
            "value": 45279,
            "unit": "ns/op",
            "extra": "52886 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/json - B/op",
            "value": 10982,
            "unit": "B/op",
            "extra": "52886 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "52886 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/msgpack",
            "value": 55604,
            "unit": "ns/op\t   32852 B/op\t      11 allocs/op",
            "extra": "43062 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/msgpack - ns/op",
            "value": 55604,
            "unit": "ns/op",
            "extra": "43062 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/msgpack - B/op",
            "value": 32852,
            "unit": "B/op",
            "extra": "43062 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/msgpack - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "43062 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_fast",
            "value": 7437,
            "unit": "ns/op\t    6979 B/op\t       3 allocs/op",
            "extra": "317911 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_fast - ns/op",
            "value": 7437,
            "unit": "ns/op",
            "extra": "317911 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_fast - B/op",
            "value": 6979,
            "unit": "B/op",
            "extra": "317911 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "317911 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_qpack",
            "value": 2801,
            "unit": "ns/op\t    2496 B/op\t       3 allocs/op",
            "extra": "829837 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_qpack - ns/op",
            "value": 2801,
            "unit": "ns/op",
            "extra": "829837 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_qpack - B/op",
            "value": 2496,
            "unit": "B/op",
            "extra": "829837 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_qpack - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "829837 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_dense",
            "value": 2898,
            "unit": "ns/op\t    2496 B/op\t       3 allocs/op",
            "extra": "837019 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_dense - ns/op",
            "value": 2898,
            "unit": "ns/op",
            "extra": "837019 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_dense - B/op",
            "value": 2496,
            "unit": "B/op",
            "extra": "837019 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "837019 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/json",
            "value": 211021,
            "unit": "ns/op\t   21288 B/op\t      41 allocs/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/json - ns/op",
            "value": 211021,
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
            "value": 80173,
            "unit": "ns/op\t   21427 B/op\t      22 allocs/op",
            "extra": "30048 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/msgpack - ns/op",
            "value": 80173,
            "unit": "ns/op",
            "extra": "30048 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/msgpack - B/op",
            "value": 21427,
            "unit": "B/op",
            "extra": "30048 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/msgpack - allocs/op",
            "value": 22,
            "unit": "allocs/op",
            "extra": "30048 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_fast",
            "value": 14218,
            "unit": "ns/op\t   10593 B/op\t       5 allocs/op",
            "extra": "167257 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_fast - ns/op",
            "value": 14218,
            "unit": "ns/op",
            "extra": "167257 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_fast - B/op",
            "value": 10593,
            "unit": "B/op",
            "extra": "167257 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_fast - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "167257 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_qpack",
            "value": 3384,
            "unit": "ns/op\t   10595 B/op\t       5 allocs/op",
            "extra": "712597 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_qpack - ns/op",
            "value": 3384,
            "unit": "ns/op",
            "extra": "712597 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_qpack - B/op",
            "value": 10595,
            "unit": "B/op",
            "extra": "712597 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_qpack - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "712597 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_dense",
            "value": 3628,
            "unit": "ns/op\t   10676 B/op\t       7 allocs/op",
            "extra": "681218 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_dense - ns/op",
            "value": 3628,
            "unit": "ns/op",
            "extra": "681218 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_dense - B/op",
            "value": 10676,
            "unit": "B/op",
            "extra": "681218 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_dense - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "681218 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/json",
            "value": 3687,
            "unit": "ns/op\t     488 B/op\t      12 allocs/op",
            "extra": "635352 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/json - ns/op",
            "value": 3687,
            "unit": "ns/op",
            "extra": "635352 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/json - B/op",
            "value": 488,
            "unit": "B/op",
            "extra": "635352 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/json - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "635352 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/msgpack",
            "value": 1161,
            "unit": "ns/op\t     312 B/op\t       9 allocs/op",
            "extra": "2065360 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/msgpack - ns/op",
            "value": 1161,
            "unit": "ns/op",
            "extra": "2065360 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/msgpack - B/op",
            "value": 312,
            "unit": "B/op",
            "extra": "2065360 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/msgpack - allocs/op",
            "value": 9,
            "unit": "allocs/op",
            "extra": "2065360 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/qdf_fast",
            "value": 542.6,
            "unit": "ns/op\t     264 B/op\t       8 allocs/op",
            "extra": "4446255 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/qdf_fast - ns/op",
            "value": 542.6,
            "unit": "ns/op",
            "extra": "4446255 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/qdf_fast - B/op",
            "value": 264,
            "unit": "B/op",
            "extra": "4446255 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/qdf_fast - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "4446255 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/json",
            "value": 1648,
            "unit": "ns/op\t     488 B/op\t      12 allocs/op",
            "extra": "1454028 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/json - ns/op",
            "value": 1648,
            "unit": "ns/op",
            "extra": "1454028 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/json - B/op",
            "value": 488,
            "unit": "B/op",
            "extra": "1454028 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/json - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "1454028 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/msgpack",
            "value": 558.2,
            "unit": "ns/op\t     312 B/op\t       9 allocs/op",
            "extra": "4314758 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/msgpack - ns/op",
            "value": 558.2,
            "unit": "ns/op",
            "extra": "4314758 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/msgpack - B/op",
            "value": 312,
            "unit": "B/op",
            "extra": "4314758 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/msgpack - allocs/op",
            "value": 9,
            "unit": "allocs/op",
            "extra": "4314758 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/qdf_fast",
            "value": 279.9,
            "unit": "ns/op\t     264 B/op\t       8 allocs/op",
            "extra": "8721751 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/qdf_fast - ns/op",
            "value": 279.9,
            "unit": "ns/op",
            "extra": "8721751 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/qdf_fast - B/op",
            "value": 264,
            "unit": "B/op",
            "extra": "8721751 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/qdf_fast - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "8721751 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense",
            "value": 396453,
            "unit": "ns/op\t 468.38 MB/s\t  376006 B/op\t    2011 allocs/op",
            "extra": "5794 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense - ns/op",
            "value": 396453,
            "unit": "ns/op",
            "extra": "5794 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense - MB/s",
            "value": 468.38,
            "unit": "MB/s",
            "extra": "5794 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense - B/op",
            "value": 376006,
            "unit": "B/op",
            "extra": "5794 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense - allocs/op",
            "value": 2011,
            "unit": "allocs/op",
            "extra": "5794 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/json",
            "value": 2229,
            "unit": "ns/op\t     800 B/op\t      14 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/json - ns/op",
            "value": 2229,
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
            "value": 1929,
            "unit": "ns/op\t    1008 B/op\t      17 allocs/op",
            "extra": "1245295 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/msgpack - ns/op",
            "value": 1929,
            "unit": "ns/op",
            "extra": "1245295 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/msgpack - B/op",
            "value": 1008,
            "unit": "B/op",
            "extra": "1245295 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/msgpack - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "1245295 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/qdf_fast",
            "value": 1583,
            "unit": "ns/op\t     697 B/op\t      13 allocs/op",
            "extra": "1512970 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/qdf_fast - ns/op",
            "value": 1583,
            "unit": "ns/op",
            "extra": "1512970 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/qdf_fast - B/op",
            "value": 697,
            "unit": "B/op",
            "extra": "1512970 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/qdf_fast - allocs/op",
            "value": 13,
            "unit": "allocs/op",
            "extra": "1512970 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/json",
            "value": 1032,
            "unit": "ns/op\t     364 B/op\t       5 allocs/op",
            "extra": "2318287 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/json - ns/op",
            "value": 1032,
            "unit": "ns/op",
            "extra": "2318287 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/json - B/op",
            "value": 364,
            "unit": "B/op",
            "extra": "2318287 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/json - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "2318287 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/msgpack",
            "value": 1034,
            "unit": "ns/op\t     538 B/op\t       7 allocs/op",
            "extra": "2323557 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/msgpack - ns/op",
            "value": 1034,
            "unit": "ns/op",
            "extra": "2323557 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/msgpack - B/op",
            "value": 538,
            "unit": "B/op",
            "extra": "2323557 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/msgpack - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "2323557 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/qdf_fast",
            "value": 798.1,
            "unit": "ns/op\t     388 B/op\t       5 allocs/op",
            "extra": "3001966 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/qdf_fast - ns/op",
            "value": 798.1,
            "unit": "ns/op",
            "extra": "3001966 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/qdf_fast - B/op",
            "value": 388,
            "unit": "B/op",
            "extra": "3001966 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/qdf_fast - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "3001966 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/json",
            "value": 275219,
            "unit": "ns/op\t  122163 B/op\t       3 allocs/op",
            "extra": "8019 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/json - ns/op",
            "value": 275219,
            "unit": "ns/op",
            "extra": "8019 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/json - B/op",
            "value": 122163,
            "unit": "B/op",
            "extra": "8019 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/json - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "8019 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/msgpack",
            "value": 239154,
            "unit": "ns/op\t  191584 B/op\t      10 allocs/op",
            "extra": "8365 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/msgpack - ns/op",
            "value": 239154,
            "unit": "ns/op",
            "extra": "8365 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/msgpack - B/op",
            "value": 191584,
            "unit": "B/op",
            "extra": "8365 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/msgpack - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "8365 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/qdf_fast",
            "value": 88210,
            "unit": "ns/op\t   91125 B/op\t       4 allocs/op",
            "extra": "27225 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/qdf_fast - ns/op",
            "value": 88210,
            "unit": "ns/op",
            "extra": "27225 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/qdf_fast - B/op",
            "value": 91125,
            "unit": "B/op",
            "extra": "27225 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/qdf_fast - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "27225 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/json",
            "value": 1106,
            "unit": "ns/op\t     800 B/op\t      14 allocs/op",
            "extra": "2173398 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/json - ns/op",
            "value": 1106,
            "unit": "ns/op",
            "extra": "2173398 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/json - B/op",
            "value": 800,
            "unit": "B/op",
            "extra": "2173398 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/json - allocs/op",
            "value": 14,
            "unit": "allocs/op",
            "extra": "2173398 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/msgpack",
            "value": 1010,
            "unit": "ns/op\t    1008 B/op\t      17 allocs/op",
            "extra": "2395858 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/msgpack - ns/op",
            "value": 1010,
            "unit": "ns/op",
            "extra": "2395858 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/msgpack - B/op",
            "value": 1008,
            "unit": "B/op",
            "extra": "2395858 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/msgpack - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "2395858 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/qdf_fast",
            "value": 801.4,
            "unit": "ns/op\t     697 B/op\t      13 allocs/op",
            "extra": "2962702 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/qdf_fast - ns/op",
            "value": 801.4,
            "unit": "ns/op",
            "extra": "2962702 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/qdf_fast - B/op",
            "value": 697,
            "unit": "B/op",
            "extra": "2962702 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/qdf_fast - allocs/op",
            "value": 13,
            "unit": "allocs/op",
            "extra": "2962702 times\n4 procs"
          }
        ]
      },
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
          "id": "35d8f44d9167c2bab523e60c293b9a4f3b8b8a79",
          "message": "qpack: general AVX2 VPSRLVQ decode for all FOR widths 1-14 (incl. odd)\n\nThe fixed-width kernels covered 8/10/12/14/16/20/32; the remaining small\nwidths — notably the common odd FOR-delta widths 9/11/13 — still ran the\nscalar sliding window (~2 GB/s). Add a general b<=14 kernel: four values\nfit one 64-bit window even at the worst byte offset (7+4*14 < 64), so each\ngroup loads 8 bytes at its start byte, VPBROADCASTQ to all lanes, and\nVPSRLVQ by a per-group shift vector selected from a small table indexed by\nthe in-byte offset (0..7). bitUnpackU64LE now routes every width <=14 not\nalready special-cased through it; the scalar tail resumes on a byte-aligned\nboundary (groups trimmed to keep 4*groups*b a whole byte count).\n\nOutput bit-identical to scalar (parity across widths 5/6/7/9/11/13 and many\nsizes under both build configs; golden unchanged). Opt-in behind qdf_simd.\n\nMicrobench (1024 elems, -count=5, i7-9750H), scalar -> SIMD:\n  width  9: ~3700 -> ~480 ns  (~7.7x)\n  width 11: ~4200 -> ~490 ns  (~8.6x)\n  width 13: ~5200 -> ~510 ns  (~10x)\n\nDocs: USAGE/GUIDE/README SIMD coverage updated to 'every width 1-14 plus\n16/20/32' for decode.",
          "timestamp": "2026-05-29T14:17:59+03:00",
          "tree_id": "59ed41bdb48db31eea41f39f7ca0513c5e2e3148",
          "url": "https://github.com/alex60217101990/qdf/commit/35d8f44d9167c2bab523e60c293b9a4f3b8b8a79"
        },
        "date": 1780053984674,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json",
            "value": 163.2,
            "unit": "ns/op\t 153.14 MB/s\t      24 B/op\t       1 allocs/op",
            "extra": "14611197 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json - ns/op",
            "value": 163.2,
            "unit": "ns/op",
            "extra": "14611197 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json - MB/s",
            "value": 153.14,
            "unit": "MB/s",
            "extra": "14611197 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json - B/op",
            "value": 24,
            "unit": "B/op",
            "extra": "14611197 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "14611197 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal",
            "value": 190.6,
            "unit": "ns/op\t 125.93 MB/s\t      48 B/op\t       2 allocs/op",
            "extra": "12501529 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal - ns/op",
            "value": 190.6,
            "unit": "ns/op",
            "extra": "12501529 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal - MB/s",
            "value": 125.93,
            "unit": "MB/s",
            "extra": "12501529 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal - B/op",
            "value": 48,
            "unit": "B/op",
            "extra": "12501529 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "12501529 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack",
            "value": 244.9,
            "unit": "ns/op\t  65.32 MB/s\t     136 B/op\t       3 allocs/op",
            "extra": "9755014 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack - ns/op",
            "value": 244.9,
            "unit": "ns/op",
            "extra": "9755014 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack - MB/s",
            "value": 65.32,
            "unit": "MB/s",
            "extra": "9755014 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack - B/op",
            "value": 136,
            "unit": "B/op",
            "extra": "9755014 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "9755014 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast",
            "value": 313.1,
            "unit": "ns/op\t  70.26 MB/s\t      72 B/op\t       3 allocs/op",
            "extra": "7673510 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast - ns/op",
            "value": 313.1,
            "unit": "ns/op",
            "extra": "7673510 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast - MB/s",
            "value": 70.26,
            "unit": "MB/s",
            "extra": "7673510 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast - B/op",
            "value": 72,
            "unit": "B/op",
            "extra": "7673510 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "7673510 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense",
            "value": 393,
            "unit": "ns/op\t  63.62 MB/s\t      80 B/op\t       3 allocs/op",
            "extra": "6110130 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense - ns/op",
            "value": 393,
            "unit": "ns/op",
            "extra": "6110130 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense - MB/s",
            "value": 63.62,
            "unit": "MB/s",
            "extra": "6110130 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "6110130 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "6110130 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json",
            "value": 1023,
            "unit": "ns/op\t 206.16 MB/s\t     192 B/op\t       1 allocs/op",
            "extra": "2366388 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json - ns/op",
            "value": 1023,
            "unit": "ns/op",
            "extra": "2366388 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json - MB/s",
            "value": 206.16,
            "unit": "MB/s",
            "extra": "2366388 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json - B/op",
            "value": 192,
            "unit": "B/op",
            "extra": "2366388 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2366388 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal",
            "value": 1076,
            "unit": "ns/op\t 195.19 MB/s\t     416 B/op\t       2 allocs/op",
            "extra": "2234181 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal - ns/op",
            "value": 1076,
            "unit": "ns/op",
            "extra": "2234181 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal - MB/s",
            "value": 195.19,
            "unit": "MB/s",
            "extra": "2234181 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal - B/op",
            "value": 416,
            "unit": "B/op",
            "extra": "2234181 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "2234181 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack",
            "value": 1033,
            "unit": "ns/op\t 129.67 MB/s\t     688 B/op\t       5 allocs/op",
            "extra": "2332554 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack - ns/op",
            "value": 1033,
            "unit": "ns/op",
            "extra": "2332554 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack - MB/s",
            "value": 129.67,
            "unit": "MB/s",
            "extra": "2332554 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack - B/op",
            "value": 688,
            "unit": "B/op",
            "extra": "2332554 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "2332554 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast",
            "value": 614.2,
            "unit": "ns/op\t 214.93 MB/s\t     528 B/op\t       3 allocs/op",
            "extra": "3631174 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast - ns/op",
            "value": 614.2,
            "unit": "ns/op",
            "extra": "3631174 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast - MB/s",
            "value": 214.93,
            "unit": "MB/s",
            "extra": "3631174 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast - B/op",
            "value": 528,
            "unit": "B/op",
            "extra": "3631174 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3631174 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense",
            "value": 774.7,
            "unit": "ns/op\t 178.12 MB/s\t     528 B/op\t       3 allocs/op",
            "extra": "3109005 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense - ns/op",
            "value": 774.7,
            "unit": "ns/op",
            "extra": "3109005 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense - MB/s",
            "value": 178.12,
            "unit": "MB/s",
            "extra": "3109005 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense - B/op",
            "value": 528,
            "unit": "B/op",
            "extra": "3109005 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3109005 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json",
            "value": 455.6,
            "unit": "ns/op\t 228.30 MB/s\t      80 B/op\t       1 allocs/op",
            "extra": "5256771 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json - ns/op",
            "value": 455.6,
            "unit": "ns/op",
            "extra": "5256771 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json - MB/s",
            "value": 228.3,
            "unit": "MB/s",
            "extra": "5256771 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "5256771 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "5256771 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal",
            "value": 487.7,
            "unit": "ns/op\t 211.20 MB/s\t     192 B/op\t       2 allocs/op",
            "extra": "4865737 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal - ns/op",
            "value": 487.7,
            "unit": "ns/op",
            "extra": "4865737 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal - MB/s",
            "value": 211.2,
            "unit": "MB/s",
            "extra": "4865737 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal - B/op",
            "value": 192,
            "unit": "B/op",
            "extra": "4865737 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "4865737 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack",
            "value": 735.6,
            "unit": "ns/op\t 103.32 MB/s\t     320 B/op\t       4 allocs/op",
            "extra": "3264106 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack - ns/op",
            "value": 735.6,
            "unit": "ns/op",
            "extra": "3264106 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack - MB/s",
            "value": 103.32,
            "unit": "MB/s",
            "extra": "3264106 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack - B/op",
            "value": 320,
            "unit": "B/op",
            "extra": "3264106 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "3264106 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast",
            "value": 454.6,
            "unit": "ns/op\t 189.19 MB/s\t     256 B/op\t       3 allocs/op",
            "extra": "5238663 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast - ns/op",
            "value": 454.6,
            "unit": "ns/op",
            "extra": "5238663 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast - MB/s",
            "value": 189.19,
            "unit": "MB/s",
            "extra": "5238663 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast - B/op",
            "value": 256,
            "unit": "B/op",
            "extra": "5238663 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "5238663 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense",
            "value": 661.8,
            "unit": "ns/op\t 145.06 MB/s\t     256 B/op\t       3 allocs/op",
            "extra": "3616062 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense - ns/op",
            "value": 661.8,
            "unit": "ns/op",
            "extra": "3616062 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense - MB/s",
            "value": 145.06,
            "unit": "MB/s",
            "extra": "3616062 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense - B/op",
            "value": 256,
            "unit": "B/op",
            "extra": "3616062 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3616062 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json",
            "value": 1455,
            "unit": "ns/op\t 164.97 MB/s\t       0 B/op\t       0 allocs/op",
            "extra": "1651122 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json - ns/op",
            "value": 1455,
            "unit": "ns/op",
            "extra": "1651122 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json - MB/s",
            "value": 164.97,
            "unit": "MB/s",
            "extra": "1651122 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1651122 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1651122 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal",
            "value": 1530,
            "unit": "ns/op\t 156.18 MB/s\t     240 B/op\t       1 allocs/op",
            "extra": "1580619 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal - ns/op",
            "value": 1530,
            "unit": "ns/op",
            "extra": "1580619 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal - MB/s",
            "value": 156.18,
            "unit": "MB/s",
            "extra": "1580619 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal - B/op",
            "value": 240,
            "unit": "B/op",
            "extra": "1580619 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "1580619 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack",
            "value": 2972,
            "unit": "ns/op\t  46.76 MB/s\t     752 B/op\t      20 allocs/op",
            "extra": "783130 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack - ns/op",
            "value": 2972,
            "unit": "ns/op",
            "extra": "783130 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack - MB/s",
            "value": 46.76,
            "unit": "MB/s",
            "extra": "783130 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack - B/op",
            "value": 752,
            "unit": "B/op",
            "extra": "783130 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "783130 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast",
            "value": 755.1,
            "unit": "ns/op\t 219.83 MB/s\t     176 B/op\t       1 allocs/op",
            "extra": "3183031 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast - ns/op",
            "value": 755.1,
            "unit": "ns/op",
            "extra": "3183031 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast - MB/s",
            "value": 219.83,
            "unit": "MB/s",
            "extra": "3183031 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast - B/op",
            "value": 176,
            "unit": "B/op",
            "extra": "3183031 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "3183031 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense",
            "value": 676.9,
            "unit": "ns/op\t  93.07 MB/s\t      64 B/op\t       1 allocs/op",
            "extra": "3508072 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense - ns/op",
            "value": 676.9,
            "unit": "ns/op",
            "extra": "3508072 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense - MB/s",
            "value": 93.07,
            "unit": "MB/s",
            "extra": "3508072 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense - B/op",
            "value": 64,
            "unit": "B/op",
            "extra": "3508072 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "3508072 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json",
            "value": 874526,
            "unit": "ns/op\t 243.45 MB/s\t     291 B/op\t       1 allocs/op",
            "extra": "2763 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json - ns/op",
            "value": 874526,
            "unit": "ns/op",
            "extra": "2763 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json - MB/s",
            "value": 243.45,
            "unit": "MB/s",
            "extra": "2763 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json - B/op",
            "value": 291,
            "unit": "B/op",
            "extra": "2763 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2763 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal",
            "value": 900947,
            "unit": "ns/op\t 236.31 MB/s\t  213248 B/op\t       2 allocs/op",
            "extra": "2625 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal - ns/op",
            "value": 900947,
            "unit": "ns/op",
            "extra": "2625 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal - MB/s",
            "value": 236.31,
            "unit": "MB/s",
            "extra": "2625 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal - B/op",
            "value": 213248,
            "unit": "B/op",
            "extra": "2625 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "2625 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack",
            "value": 777186,
            "unit": "ns/op\t 174.51 MB/s\t  524380 B/op\t      15 allocs/op",
            "extra": "3008 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack - ns/op",
            "value": 777186,
            "unit": "ns/op",
            "extra": "3008 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack - MB/s",
            "value": 174.51,
            "unit": "MB/s",
            "extra": "3008 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack - B/op",
            "value": 524380,
            "unit": "B/op",
            "extra": "3008 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "3008 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast",
            "value": 243718,
            "unit": "ns/op\t 527.79 MB/s\t  131173 B/op\t       3 allocs/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast - ns/op",
            "value": 243718,
            "unit": "ns/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast - MB/s",
            "value": 527.79,
            "unit": "MB/s",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast - B/op",
            "value": 131173,
            "unit": "B/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense",
            "value": 160753,
            "unit": "ns/op\t 233.71 MB/s\t   42332 B/op\t      10 allocs/op",
            "extra": "14847 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense - ns/op",
            "value": 160753,
            "unit": "ns/op",
            "extra": "14847 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense - MB/s",
            "value": 233.71,
            "unit": "MB/s",
            "extra": "14847 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense - B/op",
            "value": 42332,
            "unit": "B/op",
            "extra": "14847 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "14847 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json",
            "value": 916979,
            "unit": "ns/op\t 269.26 MB/s\t   48350 B/op\t    1001 allocs/op",
            "extra": "2602 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json - ns/op",
            "value": 916979,
            "unit": "ns/op",
            "extra": "2602 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json - MB/s",
            "value": 269.26,
            "unit": "MB/s",
            "extra": "2602 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json - B/op",
            "value": 48350,
            "unit": "B/op",
            "extra": "2602 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json - allocs/op",
            "value": 1001,
            "unit": "allocs/op",
            "extra": "2602 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal",
            "value": 955522,
            "unit": "ns/op\t 258.39 MB/s\t  303338 B/op\t    1002 allocs/op",
            "extra": "2497 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal - ns/op",
            "value": 955522,
            "unit": "ns/op",
            "extra": "2497 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal - MB/s",
            "value": 258.39,
            "unit": "MB/s",
            "extra": "2497 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal - B/op",
            "value": 303338,
            "unit": "B/op",
            "extra": "2497 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal - allocs/op",
            "value": 1002,
            "unit": "allocs/op",
            "extra": "2497 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack",
            "value": 586151,
            "unit": "ns/op\t 316.71 MB/s\t  548384 B/op\t    1015 allocs/op",
            "extra": "4093 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack - ns/op",
            "value": 586151,
            "unit": "ns/op",
            "extra": "4093 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack - MB/s",
            "value": 316.71,
            "unit": "MB/s",
            "extra": "4093 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack - B/op",
            "value": 548384,
            "unit": "B/op",
            "extra": "4093 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack - allocs/op",
            "value": 1015,
            "unit": "allocs/op",
            "extra": "4093 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast",
            "value": 182497,
            "unit": "ns/op\t1017.27 MB/s\t  188905 B/op\t       3 allocs/op",
            "extra": "13155 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast - ns/op",
            "value": 182497,
            "unit": "ns/op",
            "extra": "13155 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast - MB/s",
            "value": 1017.27,
            "unit": "MB/s",
            "extra": "13155 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast - B/op",
            "value": 188905,
            "unit": "B/op",
            "extra": "13155 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "13155 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense",
            "value": 182772,
            "unit": "ns/op\t1015.74 MB/s\t  189044 B/op\t       3 allocs/op",
            "extra": "13107 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense - ns/op",
            "value": 182772,
            "unit": "ns/op",
            "extra": "13107 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense - MB/s",
            "value": 1015.74,
            "unit": "MB/s",
            "extra": "13107 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense - B/op",
            "value": 189044,
            "unit": "B/op",
            "extra": "13107 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "13107 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json",
            "value": 757.2,
            "unit": "ns/op\t  31.69 MB/s\t     248 B/op\t       6 allocs/op",
            "extra": "3164817 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json - ns/op",
            "value": 757.2,
            "unit": "ns/op",
            "extra": "3164817 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json - MB/s",
            "value": 31.69,
            "unit": "MB/s",
            "extra": "3164817 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json - B/op",
            "value": 248,
            "unit": "B/op",
            "extra": "3164817 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "3164817 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack",
            "value": 313.3,
            "unit": "ns/op\t  51.07 MB/s\t      77 B/op\t       3 allocs/op",
            "extra": "7658079 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack - ns/op",
            "value": 313.3,
            "unit": "ns/op",
            "extra": "7658079 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack - MB/s",
            "value": 51.07,
            "unit": "MB/s",
            "extra": "7658079 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack - B/op",
            "value": 77,
            "unit": "B/op",
            "extra": "7658079 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "7658079 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast",
            "value": 158.1,
            "unit": "ns/op\t 139.13 MB/s\t      29 B/op\t       2 allocs/op",
            "extra": "14891796 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast - ns/op",
            "value": 158.1,
            "unit": "ns/op",
            "extra": "14891796 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast - MB/s",
            "value": 139.13,
            "unit": "MB/s",
            "extra": "14891796 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast - B/op",
            "value": 29,
            "unit": "B/op",
            "extra": "14891796 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "14891796 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense",
            "value": 325.6,
            "unit": "ns/op\t  76.79 MB/s\t      72 B/op\t       4 allocs/op",
            "extra": "7375452 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense - ns/op",
            "value": 325.6,
            "unit": "ns/op",
            "extra": "7375452 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense - MB/s",
            "value": 76.79,
            "unit": "MB/s",
            "extra": "7375452 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense - B/op",
            "value": 72,
            "unit": "B/op",
            "extra": "7375452 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "7375452 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json",
            "value": 4592,
            "unit": "ns/op\t  45.73 MB/s\t     448 B/op\t      10 allocs/op",
            "extra": "505674 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json - ns/op",
            "value": 4592,
            "unit": "ns/op",
            "extra": "505674 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json - MB/s",
            "value": 45.73,
            "unit": "MB/s",
            "extra": "505674 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json - B/op",
            "value": 448,
            "unit": "B/op",
            "extra": "505674 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "505674 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack",
            "value": 1616,
            "unit": "ns/op\t  82.94 MB/s\t     272 B/op\t       7 allocs/op",
            "extra": "1484125 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack - ns/op",
            "value": 1616,
            "unit": "ns/op",
            "extra": "1484125 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack - MB/s",
            "value": 82.94,
            "unit": "MB/s",
            "extra": "1484125 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack - B/op",
            "value": 272,
            "unit": "B/op",
            "extra": "1484125 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "1484125 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast",
            "value": 859.9,
            "unit": "ns/op\t 153.51 MB/s\t     224 B/op\t       6 allocs/op",
            "extra": "2776640 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast - ns/op",
            "value": 859.9,
            "unit": "ns/op",
            "extra": "2776640 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast - MB/s",
            "value": 153.51,
            "unit": "MB/s",
            "extra": "2776640 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast - B/op",
            "value": 224,
            "unit": "B/op",
            "extra": "2776640 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "2776640 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense",
            "value": 1421,
            "unit": "ns/op\t  97.14 MB/s\t     624 B/op\t       8 allocs/op",
            "extra": "1692459 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense - ns/op",
            "value": 1421,
            "unit": "ns/op",
            "extra": "1692459 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense - MB/s",
            "value": 97.14,
            "unit": "MB/s",
            "extra": "1692459 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense - B/op",
            "value": 624,
            "unit": "B/op",
            "extra": "1692459 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "1692459 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json",
            "value": 2536,
            "unit": "ns/op\t  40.62 MB/s\t     664 B/op\t      15 allocs/op",
            "extra": "890343 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json - ns/op",
            "value": 2536,
            "unit": "ns/op",
            "extra": "890343 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json - MB/s",
            "value": 40.62,
            "unit": "MB/s",
            "extra": "890343 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json - B/op",
            "value": 664,
            "unit": "B/op",
            "extra": "890343 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "890343 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack",
            "value": 1094,
            "unit": "ns/op\t  69.47 MB/s\t     160 B/op\t       6 allocs/op",
            "extra": "2189406 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack - ns/op",
            "value": 1094,
            "unit": "ns/op",
            "extra": "2189406 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack - MB/s",
            "value": 69.47,
            "unit": "MB/s",
            "extra": "2189406 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack - B/op",
            "value": 160,
            "unit": "B/op",
            "extra": "2189406 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "2189406 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast",
            "value": 403.3,
            "unit": "ns/op\t 213.24 MB/s\t     112 B/op\t       5 allocs/op",
            "extra": "5906882 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast - ns/op",
            "value": 403.3,
            "unit": "ns/op",
            "extra": "5906882 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast - MB/s",
            "value": 213.24,
            "unit": "MB/s",
            "extra": "5906882 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast - B/op",
            "value": 112,
            "unit": "B/op",
            "extra": "5906882 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "5906882 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense",
            "value": 917.6,
            "unit": "ns/op\t 104.62 MB/s\t     297 B/op\t      15 allocs/op",
            "extra": "2613721 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense - ns/op",
            "value": 917.6,
            "unit": "ns/op",
            "extra": "2613721 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense - MB/s",
            "value": 104.62,
            "unit": "MB/s",
            "extra": "2613721 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense - B/op",
            "value": 297,
            "unit": "B/op",
            "extra": "2613721 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "2613721 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json",
            "value": 7127,
            "unit": "ns/op\t  33.53 MB/s\t    1200 B/op\t      29 allocs/op",
            "extra": "331864 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json - ns/op",
            "value": 7127,
            "unit": "ns/op",
            "extra": "331864 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json - MB/s",
            "value": 33.53,
            "unit": "MB/s",
            "extra": "331864 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json - B/op",
            "value": 1200,
            "unit": "B/op",
            "extra": "331864 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json - allocs/op",
            "value": 29,
            "unit": "allocs/op",
            "extra": "331864 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack",
            "value": 3902,
            "unit": "ns/op\t  35.62 MB/s\t     312 B/op\t      18 allocs/op",
            "extra": "610838 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack - ns/op",
            "value": 3902,
            "unit": "ns/op",
            "extra": "610838 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack - MB/s",
            "value": 35.62,
            "unit": "MB/s",
            "extra": "610838 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack - B/op",
            "value": 312,
            "unit": "B/op",
            "extra": "610838 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "610838 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast",
            "value": 1987,
            "unit": "ns/op\t  83.52 MB/s\t     264 B/op\t      17 allocs/op",
            "extra": "1207425 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast - ns/op",
            "value": 1987,
            "unit": "ns/op",
            "extra": "1207425 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast - MB/s",
            "value": 83.52,
            "unit": "MB/s",
            "extra": "1207425 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast - B/op",
            "value": 264,
            "unit": "B/op",
            "extra": "1207425 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "1207425 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense",
            "value": 1953,
            "unit": "ns/op\t  32.25 MB/s\t     304 B/op\t      19 allocs/op",
            "extra": "1228179 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense - ns/op",
            "value": 1953,
            "unit": "ns/op",
            "extra": "1228179 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense - MB/s",
            "value": 32.25,
            "unit": "MB/s",
            "extra": "1228179 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense - B/op",
            "value": 304,
            "unit": "B/op",
            "extra": "1228179 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense - allocs/op",
            "value": 19,
            "unit": "allocs/op",
            "extra": "1228179 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/json",
            "value": 4527937,
            "unit": "ns/op\t  47.02 MB/s\t  638352 B/op\t    5020 allocs/op",
            "extra": "526 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/json - ns/op",
            "value": 4527937,
            "unit": "ns/op",
            "extra": "526 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/json - MB/s",
            "value": 47.02,
            "unit": "MB/s",
            "extra": "526 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/json - B/op",
            "value": 638352,
            "unit": "B/op",
            "extra": "526 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/json - allocs/op",
            "value": 5020,
            "unit": "allocs/op",
            "extra": "526 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack",
            "value": 1573414,
            "unit": "ns/op\t  86.20 MB/s\t  409043 B/op\t    5007 allocs/op",
            "extra": "1514 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack - ns/op",
            "value": 1573414,
            "unit": "ns/op",
            "extra": "1514 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack - MB/s",
            "value": 86.2,
            "unit": "MB/s",
            "extra": "1514 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack - B/op",
            "value": 409043,
            "unit": "B/op",
            "extra": "1514 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack - allocs/op",
            "value": 5007,
            "unit": "allocs/op",
            "extra": "1514 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast",
            "value": 746308,
            "unit": "ns/op\t 172.36 MB/s\t  220500 B/op\t    5003 allocs/op",
            "extra": "3168 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast - ns/op",
            "value": 746308,
            "unit": "ns/op",
            "extra": "3168 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast - MB/s",
            "value": 172.36,
            "unit": "MB/s",
            "extra": "3168 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast - B/op",
            "value": 220500,
            "unit": "B/op",
            "extra": "3168 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast - allocs/op",
            "value": 5003,
            "unit": "allocs/op",
            "extra": "3168 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_dense",
            "value": 205352,
            "unit": "ns/op\t 182.95 MB/s\t  318266 B/op\t    5022 allocs/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_dense - ns/op",
            "value": 205352,
            "unit": "ns/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_dense - MB/s",
            "value": 182.95,
            "unit": "MB/s",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_dense - B/op",
            "value": 318266,
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
            "value": 3440638,
            "unit": "ns/op\t  71.76 MB/s\t  442536 B/op\t    7019 allocs/op",
            "extra": "697 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/json - ns/op",
            "value": 3440638,
            "unit": "ns/op",
            "extra": "697 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/json - MB/s",
            "value": 71.76,
            "unit": "MB/s",
            "extra": "697 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/json - B/op",
            "value": 442536,
            "unit": "B/op",
            "extra": "697 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/json - allocs/op",
            "value": 7019,
            "unit": "allocs/op",
            "extra": "697 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack",
            "value": 1091554,
            "unit": "ns/op\t 170.07 MB/s\t  407514 B/op\t    7007 allocs/op",
            "extra": "2172 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack - ns/op",
            "value": 1091554,
            "unit": "ns/op",
            "extra": "2172 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack - MB/s",
            "value": 170.07,
            "unit": "MB/s",
            "extra": "2172 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack - B/op",
            "value": 407514,
            "unit": "B/op",
            "extra": "2172 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack - allocs/op",
            "value": 7007,
            "unit": "allocs/op",
            "extra": "2172 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast",
            "value": 424178,
            "unit": "ns/op\t 437.67 MB/s\t  251713 B/op\t    7002 allocs/op",
            "extra": "5596 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast - ns/op",
            "value": 424178,
            "unit": "ns/op",
            "extra": "5596 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast - MB/s",
            "value": 437.67,
            "unit": "MB/s",
            "extra": "5596 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast - B/op",
            "value": 251713,
            "unit": "B/op",
            "extra": "5596 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast - allocs/op",
            "value": 7002,
            "unit": "allocs/op",
            "extra": "5596 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense",
            "value": 424685,
            "unit": "ns/op\t 437.15 MB/s\t  255169 B/op\t    7005 allocs/op",
            "extra": "5554 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense - ns/op",
            "value": 424685,
            "unit": "ns/op",
            "extra": "5554 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense - MB/s",
            "value": 437.15,
            "unit": "MB/s",
            "extra": "5554 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense - B/op",
            "value": 255169,
            "unit": "B/op",
            "extra": "5554 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense - allocs/op",
            "value": 7005,
            "unit": "allocs/op",
            "extra": "5554 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen",
            "value": 323849,
            "unit": "ns/op\t 573.26 MB/s\t  908027 B/op\t      26 allocs/op",
            "extra": "7117 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen - ns/op",
            "value": 323849,
            "unit": "ns/op",
            "extra": "7117 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen - MB/s",
            "value": 573.26,
            "unit": "MB/s",
            "extra": "7117 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen - B/op",
            "value": 908027,
            "unit": "B/op",
            "extra": "7117 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen - allocs/op",
            "value": 26,
            "unit": "allocs/op",
            "extra": "7117 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen",
            "value": 422448,
            "unit": "ns/op\t 439.46 MB/s\t  251648 B/op\t    7001 allocs/op",
            "extra": "5480 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen - ns/op",
            "value": 422448,
            "unit": "ns/op",
            "extra": "5480 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen - MB/s",
            "value": 439.46,
            "unit": "MB/s",
            "extra": "5480 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen - B/op",
            "value": 251648,
            "unit": "B/op",
            "extra": "5480 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen - allocs/op",
            "value": 7001,
            "unit": "allocs/op",
            "extra": "5480 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json",
            "value": 142689,
            "unit": "ns/op\t 188.70 MB/s\t   27435 B/op\t       2 allocs/op",
            "extra": "16816 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json - ns/op",
            "value": 142689,
            "unit": "ns/op",
            "extra": "16816 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json - MB/s",
            "value": 188.7,
            "unit": "MB/s",
            "extra": "16816 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json - B/op",
            "value": 27435,
            "unit": "B/op",
            "extra": "16816 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "16816 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack",
            "value": 173839,
            "unit": "ns/op\t 218.56 MB/s\t  131235 B/op\t      13 allocs/op",
            "extra": "13774 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack - ns/op",
            "value": 173839,
            "unit": "ns/op",
            "extra": "13774 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack - MB/s",
            "value": 218.56,
            "unit": "MB/s",
            "extra": "13774 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack - B/op",
            "value": 131235,
            "unit": "B/op",
            "extra": "13774 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack - allocs/op",
            "value": 13,
            "unit": "allocs/op",
            "extra": "13774 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf",
            "value": 15086,
            "unit": "ns/op\t 580.61 MB/s\t    9796 B/op\t       3 allocs/op",
            "extra": "159949 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf - ns/op",
            "value": 15086,
            "unit": "ns/op",
            "extra": "159949 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf - MB/s",
            "value": 580.61,
            "unit": "MB/s",
            "extra": "159949 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf - B/op",
            "value": 9796,
            "unit": "B/op",
            "extra": "159949 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "159949 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json",
            "value": 635383,
            "unit": "ns/op\t  42.38 MB/s\t  104576 B/op\t      65 allocs/op",
            "extra": "3754 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json - ns/op",
            "value": 635383,
            "unit": "ns/op",
            "extra": "3754 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json - MB/s",
            "value": 42.38,
            "unit": "MB/s",
            "extra": "3754 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json - B/op",
            "value": 104576,
            "unit": "B/op",
            "extra": "3754 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json - allocs/op",
            "value": 65,
            "unit": "allocs/op",
            "extra": "3754 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack",
            "value": 247134,
            "unit": "ns/op\t 153.74 MB/s\t   68194 B/op\t      29 allocs/op",
            "extra": "9325 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack - ns/op",
            "value": 247134,
            "unit": "ns/op",
            "extra": "9325 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack - MB/s",
            "value": 153.74,
            "unit": "MB/s",
            "extra": "9325 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack - B/op",
            "value": 68194,
            "unit": "B/op",
            "extra": "9325 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack - allocs/op",
            "value": 29,
            "unit": "allocs/op",
            "extra": "9325 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf",
            "value": 12727,
            "unit": "ns/op\t 688.24 MB/s\t   42334 B/op\t      11 allocs/op",
            "extra": "193354 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf - ns/op",
            "value": 12727,
            "unit": "ns/op",
            "extra": "193354 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf - MB/s",
            "value": 688.24,
            "unit": "MB/s",
            "extra": "193354 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf - B/op",
            "value": 42334,
            "unit": "B/op",
            "extra": "193354 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "193354 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json",
            "value": 71383,
            "unit": "ns/op\t 242.55 MB/s\t   18532 B/op\t       2 allocs/op",
            "extra": "33532 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json - ns/op",
            "value": 71383,
            "unit": "ns/op",
            "extra": "33532 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json - MB/s",
            "value": 242.55,
            "unit": "MB/s",
            "extra": "33532 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json - B/op",
            "value": 18532,
            "unit": "B/op",
            "extra": "33532 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "33532 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack",
            "value": 106694,
            "unit": "ns/op\t 259.71 MB/s\t   65625 B/op\t      12 allocs/op",
            "extra": "22466 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack - ns/op",
            "value": 106694,
            "unit": "ns/op",
            "extra": "22466 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack - MB/s",
            "value": 259.71,
            "unit": "MB/s",
            "extra": "22466 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack - B/op",
            "value": 65625,
            "unit": "B/op",
            "extra": "22466 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "22466 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf",
            "value": 12235,
            "unit": "ns/op\t  46.02 MB/s\t     768 B/op\t       3 allocs/op",
            "extra": "193966 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf - ns/op",
            "value": 12235,
            "unit": "ns/op",
            "extra": "193966 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf - MB/s",
            "value": 46.02,
            "unit": "MB/s",
            "extra": "193966 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf - B/op",
            "value": 768,
            "unit": "B/op",
            "extra": "193966 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "193966 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json",
            "value": 392883,
            "unit": "ns/op\t  44.07 MB/s\t   75976 B/op\t      43 allocs/op",
            "extra": "6070 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json - ns/op",
            "value": 392883,
            "unit": "ns/op",
            "extra": "6070 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json - MB/s",
            "value": 44.07,
            "unit": "MB/s",
            "extra": "6070 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json - B/op",
            "value": 75976,
            "unit": "B/op",
            "extra": "6070 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json - allocs/op",
            "value": 43,
            "unit": "allocs/op",
            "extra": "6070 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack",
            "value": 162678,
            "unit": "ns/op\t 170.34 MB/s\t   49543 B/op\t      18 allocs/op",
            "extra": "14731 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack - ns/op",
            "value": 162678,
            "unit": "ns/op",
            "extra": "14731 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack - MB/s",
            "value": 170.34,
            "unit": "MB/s",
            "extra": "14731 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack - B/op",
            "value": 49543,
            "unit": "B/op",
            "extra": "14731 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "14731 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf",
            "value": 10350,
            "unit": "ns/op\t  54.40 MB/s\t   32895 B/op\t       6 allocs/op",
            "extra": "230457 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf - ns/op",
            "value": 10350,
            "unit": "ns/op",
            "extra": "230457 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf - MB/s",
            "value": 54.4,
            "unit": "MB/s",
            "extra": "230457 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf - B/op",
            "value": 32895,
            "unit": "B/op",
            "extra": "230457 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "230457 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json",
            "value": 28414,
            "unit": "ns/op\t 240.27 MB/s\t    6962 B/op\t       2 allocs/op",
            "extra": "84092 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json - ns/op",
            "value": 28414,
            "unit": "ns/op",
            "extra": "84092 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json - MB/s",
            "value": 240.27,
            "unit": "MB/s",
            "extra": "84092 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json - B/op",
            "value": 6962,
            "unit": "B/op",
            "extra": "84092 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "84092 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack",
            "value": 38276,
            "unit": "ns/op\t 241.64 MB/s\t   32804 B/op\t      11 allocs/op",
            "extra": "62344 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack - ns/op",
            "value": 38276,
            "unit": "ns/op",
            "extra": "62344 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack - MB/s",
            "value": 241.64,
            "unit": "MB/s",
            "extra": "62344 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack - B/op",
            "value": 32804,
            "unit": "B/op",
            "extra": "62344 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "62344 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf",
            "value": 11752,
            "unit": "ns/op\t  26.21 MB/s\t     416 B/op\t       3 allocs/op",
            "extra": "200983 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf - ns/op",
            "value": 11752,
            "unit": "ns/op",
            "extra": "200983 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf - MB/s",
            "value": 26.21,
            "unit": "MB/s",
            "extra": "200983 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf - B/op",
            "value": 416,
            "unit": "B/op",
            "extra": "200983 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "200983 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json",
            "value": 150278,
            "unit": "ns/op\t  45.43 MB/s\t   25504 B/op\t      19 allocs/op",
            "extra": "15956 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json - ns/op",
            "value": 150278,
            "unit": "ns/op",
            "extra": "15956 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json - MB/s",
            "value": 45.43,
            "unit": "MB/s",
            "extra": "15956 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json - B/op",
            "value": 25504,
            "unit": "B/op",
            "extra": "15956 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json - allocs/op",
            "value": 19,
            "unit": "allocs/op",
            "extra": "15956 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack",
            "value": 54454,
            "unit": "ns/op\t 169.85 MB/s\t   16570 B/op\t       8 allocs/op",
            "extra": "43904 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack - ns/op",
            "value": 54454,
            "unit": "ns/op",
            "extra": "43904 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack - MB/s",
            "value": 169.85,
            "unit": "MB/s",
            "extra": "43904 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack - B/op",
            "value": 16570,
            "unit": "B/op",
            "extra": "43904 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "43904 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf",
            "value": 5830,
            "unit": "ns/op\t  52.83 MB/s\t   16451 B/op\t       4 allocs/op",
            "extra": "400272 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf - ns/op",
            "value": 5830,
            "unit": "ns/op",
            "extra": "400272 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf - MB/s",
            "value": 52.83,
            "unit": "MB/s",
            "extra": "400272 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf - B/op",
            "value": 16451,
            "unit": "B/op",
            "extra": "400272 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "400272 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json",
            "value": 173176,
            "unit": "ns/op\t 424.16 MB/s\t   73801 B/op\t       2 allocs/op",
            "extra": "13837 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json - ns/op",
            "value": 173176,
            "unit": "ns/op",
            "extra": "13837 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json - MB/s",
            "value": 424.16,
            "unit": "MB/s",
            "extra": "13837 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json - B/op",
            "value": 73801,
            "unit": "B/op",
            "extra": "13837 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "13837 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack",
            "value": 148575,
            "unit": "ns/op\t 401.90 MB/s\t  131100 B/op\t      13 allocs/op",
            "extra": "16141 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack - ns/op",
            "value": 148575,
            "unit": "ns/op",
            "extra": "16141 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack - MB/s",
            "value": 401.9,
            "unit": "MB/s",
            "extra": "16141 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack - B/op",
            "value": 131100,
            "unit": "B/op",
            "extra": "16141 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack - allocs/op",
            "value": 13,
            "unit": "allocs/op",
            "extra": "16141 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf",
            "value": 88390,
            "unit": "ns/op\t 380.21 MB/s\t   41164 B/op\t       3 allocs/op",
            "extra": "26784 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf - ns/op",
            "value": 88390,
            "unit": "ns/op",
            "extra": "26784 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf - MB/s",
            "value": 380.21,
            "unit": "MB/s",
            "extra": "26784 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf - B/op",
            "value": 41164,
            "unit": "B/op",
            "extra": "26784 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "26784 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json",
            "value": 983461,
            "unit": "ns/op\t  74.69 MB/s\t  125256 B/op\t    2016 allocs/op",
            "extra": "2409 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json - ns/op",
            "value": 983461,
            "unit": "ns/op",
            "extra": "2409 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json - MB/s",
            "value": 74.69,
            "unit": "MB/s",
            "extra": "2409 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json - B/op",
            "value": 125256,
            "unit": "B/op",
            "extra": "2409 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json - allocs/op",
            "value": 2016,
            "unit": "allocs/op",
            "extra": "2409 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack",
            "value": 293842,
            "unit": "ns/op\t 203.21 MB/s\t  114785 B/op\t    2007 allocs/op",
            "extra": "8133 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack - ns/op",
            "value": 293842,
            "unit": "ns/op",
            "extra": "8133 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack - MB/s",
            "value": 203.21,
            "unit": "MB/s",
            "extra": "8133 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack - B/op",
            "value": 114785,
            "unit": "B/op",
            "extra": "8133 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack - allocs/op",
            "value": 2007,
            "unit": "allocs/op",
            "extra": "8133 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf",
            "value": 101085,
            "unit": "ns/op\t 332.46 MB/s\t   65198 B/op\t    1012 allocs/op",
            "extra": "23631 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf - ns/op",
            "value": 101085,
            "unit": "ns/op",
            "extra": "23631 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf - MB/s",
            "value": 332.46,
            "unit": "MB/s",
            "extra": "23631 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf - B/op",
            "value": 65198,
            "unit": "B/op",
            "extra": "23631 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf - allocs/op",
            "value": 1012,
            "unit": "allocs/op",
            "extra": "23631 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/json",
            "value": 26564,
            "unit": "ns/op\t    2736 B/op\t       2 allocs/op",
            "extra": "90060 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/json - ns/op",
            "value": 26564,
            "unit": "ns/op",
            "extra": "90060 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/json - B/op",
            "value": 2736,
            "unit": "B/op",
            "extra": "90060 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "90060 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/msgpack",
            "value": 18501,
            "unit": "ns/op\t    8225 B/op\t       9 allocs/op",
            "extra": "128498 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/msgpack - ns/op",
            "value": 18501,
            "unit": "ns/op",
            "extra": "128498 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/msgpack - B/op",
            "value": 8225,
            "unit": "B/op",
            "extra": "128498 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/msgpack - allocs/op",
            "value": 9,
            "unit": "allocs/op",
            "extra": "128498 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/qdf_fast",
            "value": 1961,
            "unit": "ns/op\t    2784 B/op\t       3 allocs/op",
            "extra": "1233528 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/qdf_fast - ns/op",
            "value": 1961,
            "unit": "ns/op",
            "extra": "1233528 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/qdf_fast - B/op",
            "value": 2784,
            "unit": "B/op",
            "extra": "1233528 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1233528 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/json",
            "value": 29726,
            "unit": "ns/op\t    2736 B/op\t       2 allocs/op",
            "extra": "80426 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/json - ns/op",
            "value": 29726,
            "unit": "ns/op",
            "extra": "80426 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/json - B/op",
            "value": 2736,
            "unit": "B/op",
            "extra": "80426 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "80426 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/msgpack",
            "value": 20407,
            "unit": "ns/op\t   16418 B/op\t      10 allocs/op",
            "extra": "118226 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/msgpack - ns/op",
            "value": 20407,
            "unit": "ns/op",
            "extra": "118226 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/msgpack - B/op",
            "value": 16418,
            "unit": "B/op",
            "extra": "118226 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/msgpack - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "118226 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/qdf_fast",
            "value": 2317,
            "unit": "ns/op\t    4961 B/op\t       3 allocs/op",
            "extra": "999454 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/qdf_fast - ns/op",
            "value": 2317,
            "unit": "ns/op",
            "extra": "999454 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/qdf_fast - B/op",
            "value": 4961,
            "unit": "B/op",
            "extra": "999454 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "999454 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/json",
            "value": 75740,
            "unit": "ns/op\t    4384 B/op\t      16 allocs/op",
            "extra": "31777 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/json - ns/op",
            "value": 75740,
            "unit": "ns/op",
            "extra": "31777 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/json - B/op",
            "value": 4384,
            "unit": "B/op",
            "extra": "31777 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/json - allocs/op",
            "value": 16,
            "unit": "allocs/op",
            "extra": "31777 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/msgpack",
            "value": 26182,
            "unit": "ns/op\t    4280 B/op\t       8 allocs/op",
            "extra": "91320 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/msgpack - ns/op",
            "value": 26182,
            "unit": "ns/op",
            "extra": "91320 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/msgpack - B/op",
            "value": 4280,
            "unit": "B/op",
            "extra": "91320 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/msgpack - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "91320 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/qdf_fast",
            "value": 3862,
            "unit": "ns/op\t    2112 B/op\t       3 allocs/op",
            "extra": "626906 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/qdf_fast - ns/op",
            "value": 3862,
            "unit": "ns/op",
            "extra": "626906 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/qdf_fast - B/op",
            "value": 2112,
            "unit": "B/op",
            "extra": "626906 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "626906 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/json",
            "value": 153141154,
            "unit": "ns/op\t 243.15 MB/s\t57130198 B/op\t  350217 allocs/op",
            "extra": "15 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/json - ns/op",
            "value": 153141154,
            "unit": "ns/op",
            "extra": "15 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/json - MB/s",
            "value": 243.15,
            "unit": "MB/s",
            "extra": "15 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/json - B/op",
            "value": 57130198,
            "unit": "B/op",
            "extra": "15 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/json - allocs/op",
            "value": 350217,
            "unit": "allocs/op",
            "extra": "15 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/msgpack",
            "value": 84540735,
            "unit": "ns/op\t 288.25 MB/s\t68709102 B/op\t  100022 allocs/op",
            "extra": "28 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/msgpack - ns/op",
            "value": 84540735,
            "unit": "ns/op",
            "extra": "28 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/msgpack - MB/s",
            "value": 288.25,
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
            "value": 27625209,
            "unit": "ns/op\t 872.67 MB/s\t29512042 B/op\t      19 allocs/op",
            "extra": "90 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_fast - ns/op",
            "value": 27625209,
            "unit": "ns/op",
            "extra": "90 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_fast - MB/s",
            "value": 872.67,
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
            "value": 24830377,
            "unit": "ns/op\t 946.23 MB/s\t28815726 B/op\t      19 allocs/op",
            "extra": "100 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_qpack - ns/op",
            "value": 24830377,
            "unit": "ns/op",
            "extra": "100 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_qpack - MB/s",
            "value": 946.23,
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
            "value": 27405704,
            "unit": "ns/op\t 659.93 MB/s\t24115014 B/op\t      74 allocs/op",
            "extra": "79 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_dense - ns/op",
            "value": 27405704,
            "unit": "ns/op",
            "extra": "79 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_dense - MB/s",
            "value": 659.93,
            "unit": "MB/s",
            "extra": "79 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_dense - B/op",
            "value": 24115014,
            "unit": "B/op",
            "extra": "79 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_dense - allocs/op",
            "value": 74,
            "unit": "allocs/op",
            "extra": "79 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/json",
            "value": 617023266,
            "unit": "ns/op\t  60.36 MB/s\t119804008 B/op\t 1559637 allocs/op",
            "extra": "4 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/json - ns/op",
            "value": 617023266,
            "unit": "ns/op",
            "extra": "4 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/json - MB/s",
            "value": 60.36,
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
            "value": 187351699,
            "unit": "ns/op\t 130.09 MB/s\t74390962 B/op\t 1425125 allocs/op",
            "extra": "12 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/msgpack - ns/op",
            "value": 187351699,
            "unit": "ns/op",
            "extra": "12 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/msgpack - MB/s",
            "value": 130.09,
            "unit": "MB/s",
            "extra": "12 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/msgpack - B/op",
            "value": 74390962,
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
            "value": 73999436,
            "unit": "ns/op\t 325.83 MB/s\t48380713 B/op\t  875099 allocs/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_fast - ns/op",
            "value": 73999436,
            "unit": "ns/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_fast - MB/s",
            "value": 325.83,
            "unit": "MB/s",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_fast - B/op",
            "value": 48380713,
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
            "value": 68954027,
            "unit": "ns/op\t 340.77 MB/s\t48381564 B/op\t  875099 allocs/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_qpack - ns/op",
            "value": 68954027,
            "unit": "ns/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_qpack - MB/s",
            "value": 340.77,
            "unit": "MB/s",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_qpack - B/op",
            "value": 48381564,
            "unit": "B/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_qpack - allocs/op",
            "value": 875099,
            "unit": "allocs/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense",
            "value": 62208758,
            "unit": "ns/op\t 290.76 MB/s\t50892395 B/op\t  790948 allocs/op",
            "extra": "37 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense - ns/op",
            "value": 62208758,
            "unit": "ns/op",
            "extra": "37 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense - MB/s",
            "value": 290.76,
            "unit": "MB/s",
            "extra": "37 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense - B/op",
            "value": 50892395,
            "unit": "B/op",
            "extra": "37 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense - allocs/op",
            "value": 790948,
            "unit": "allocs/op",
            "extra": "37 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/json",
            "value": 8665,
            "unit": "ns/op\t    3408 B/op\t      84 allocs/op",
            "extra": "284196 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/json - ns/op",
            "value": 8665,
            "unit": "ns/op",
            "extra": "284196 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/json - B/op",
            "value": 3408,
            "unit": "B/op",
            "extra": "284196 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/json - allocs/op",
            "value": 84,
            "unit": "allocs/op",
            "extra": "284196 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/msgpack",
            "value": 4539,
            "unit": "ns/op\t    1536 B/op\t      46 allocs/op",
            "extra": "542271 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/msgpack - ns/op",
            "value": 4539,
            "unit": "ns/op",
            "extra": "542271 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/msgpack - B/op",
            "value": 1536,
            "unit": "B/op",
            "extra": "542271 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/msgpack - allocs/op",
            "value": 46,
            "unit": "allocs/op",
            "extra": "542271 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_fast",
            "value": 1558,
            "unit": "ns/op\t     416 B/op\t       3 allocs/op",
            "extra": "1540051 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_fast - ns/op",
            "value": 1558,
            "unit": "ns/op",
            "extra": "1540051 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_fast - B/op",
            "value": 416,
            "unit": "B/op",
            "extra": "1540051 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1540051 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_dense",
            "value": 1809,
            "unit": "ns/op\t     416 B/op\t       3 allocs/op",
            "extra": "1324980 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_dense - ns/op",
            "value": 1809,
            "unit": "ns/op",
            "extra": "1324980 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_dense - B/op",
            "value": 416,
            "unit": "B/op",
            "extra": "1324980 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1324980 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/json",
            "value": 18085,
            "unit": "ns/op\t    4912 B/op\t     124 allocs/op",
            "extra": "132195 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/json - ns/op",
            "value": 18085,
            "unit": "ns/op",
            "extra": "132195 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/json - B/op",
            "value": 4912,
            "unit": "B/op",
            "extra": "132195 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/json - allocs/op",
            "value": 124,
            "unit": "allocs/op",
            "extra": "132195 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/msgpack",
            "value": 7864,
            "unit": "ns/op\t    3088 B/op\t     112 allocs/op",
            "extra": "309831 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/msgpack - ns/op",
            "value": 7864,
            "unit": "ns/op",
            "extra": "309831 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/msgpack - B/op",
            "value": 3088,
            "unit": "B/op",
            "extra": "309831 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/msgpack - allocs/op",
            "value": 112,
            "unit": "allocs/op",
            "extra": "309831 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/qdf_fast",
            "value": 3227,
            "unit": "ns/op\t    2355 B/op\t      31 allocs/op",
            "extra": "735998 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/qdf_fast - ns/op",
            "value": 3227,
            "unit": "ns/op",
            "extra": "735998 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/qdf_fast - B/op",
            "value": 2355,
            "unit": "B/op",
            "extra": "735998 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/qdf_fast - allocs/op",
            "value": 31,
            "unit": "allocs/op",
            "extra": "735998 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/json",
            "value": 9828,
            "unit": "ns/op\t    2820 B/op\t      71 allocs/op",
            "extra": "247849 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/json - ns/op",
            "value": 9828,
            "unit": "ns/op",
            "extra": "247849 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/json - B/op",
            "value": 2820,
            "unit": "B/op",
            "extra": "247849 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/json - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "247849 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/msgpack",
            "value": 2490,
            "unit": "ns/op\t    1487 B/op\t      46 allocs/op",
            "extra": "911743 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/msgpack - ns/op",
            "value": 2490,
            "unit": "ns/op",
            "extra": "911743 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/msgpack - B/op",
            "value": 1487,
            "unit": "B/op",
            "extra": "911743 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/msgpack - allocs/op",
            "value": 46,
            "unit": "allocs/op",
            "extra": "911743 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/qdf_fast",
            "value": 1857,
            "unit": "ns/op\t    1403 B/op\t      26 allocs/op",
            "extra": "1303177 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/qdf_fast - ns/op",
            "value": 1857,
            "unit": "ns/op",
            "extra": "1303177 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/qdf_fast - B/op",
            "value": 1403,
            "unit": "B/op",
            "extra": "1303177 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/qdf_fast - allocs/op",
            "value": 26,
            "unit": "allocs/op",
            "extra": "1303177 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/json",
            "value": 0.3563,
            "unit": "ns/op\t    442536 B/decode\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/json - ns/op",
            "value": 0.3563,
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
            "value": 0.1193,
            "unit": "ns/op\t    407522 B/decode\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/msgpack - ns/op",
            "value": 0.1193,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/msgpack - B/decode",
            "value": 407522,
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
            "value": 0.04582,
            "unit": "ns/op\t    251763 B/decode\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/qdf_fast - ns/op",
            "value": 0.04582,
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
            "value": 3686,
            "unit": "ns/op\t     790 B/op\t      37 allocs/op",
            "extra": "671599 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/json - ns/op",
            "value": 3686,
            "unit": "ns/op",
            "extra": "671599 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/json - B/op",
            "value": 790,
            "unit": "B/op",
            "extra": "671599 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/json - allocs/op",
            "value": 37,
            "unit": "allocs/op",
            "extra": "671599 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/qdf_fast",
            "value": 636,
            "unit": "ns/op\t     344 B/op\t       3 allocs/op",
            "extra": "3740005 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/qdf_fast - ns/op",
            "value": 636,
            "unit": "ns/op",
            "extra": "3740005 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/qdf_fast - B/op",
            "value": 344,
            "unit": "B/op",
            "extra": "3740005 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3740005 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json",
            "value": 671.7,
            "unit": "ns/op\t 144.42 MB/s\t     240 B/op\t       3 allocs/op",
            "extra": "3580962 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json - ns/op",
            "value": 671.7,
            "unit": "ns/op",
            "extra": "3580962 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json - MB/s",
            "value": 144.42,
            "unit": "MB/s",
            "extra": "3580962 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json - B/op",
            "value": 240,
            "unit": "B/op",
            "extra": "3580962 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3580962 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack",
            "value": 418,
            "unit": "ns/op\t 150.73 MB/s\t     192 B/op\t       3 allocs/op",
            "extra": "5725800 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack - ns/op",
            "value": 418,
            "unit": "ns/op",
            "extra": "5725800 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack - MB/s",
            "value": 150.73,
            "unit": "MB/s",
            "extra": "5725800 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack - B/op",
            "value": 192,
            "unit": "B/op",
            "extra": "5725800 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "5725800 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf",
            "value": 424.4,
            "unit": "ns/op\t 169.64 MB/s\t     240 B/op\t       3 allocs/op",
            "extra": "5635243 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf - ns/op",
            "value": 424.4,
            "unit": "ns/op",
            "extra": "5635243 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf - MB/s",
            "value": 169.64,
            "unit": "MB/s",
            "extra": "5635243 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf - B/op",
            "value": 240,
            "unit": "B/op",
            "extra": "5635243 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "5635243 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json",
            "value": 1788,
            "unit": "ns/op\t  54.25 MB/s\t     328 B/op\t       7 allocs/op",
            "extra": "1343410 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json - ns/op",
            "value": 1788,
            "unit": "ns/op",
            "extra": "1343410 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json - MB/s",
            "value": 54.25,
            "unit": "MB/s",
            "extra": "1343410 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json - B/op",
            "value": 328,
            "unit": "B/op",
            "extra": "1343410 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "1343410 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack",
            "value": 673.3,
            "unit": "ns/op\t  93.57 MB/s\t     160 B/op\t       4 allocs/op",
            "extra": "3586669 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack - ns/op",
            "value": 673.3,
            "unit": "ns/op",
            "extra": "3586669 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack - MB/s",
            "value": 93.57,
            "unit": "MB/s",
            "extra": "3586669 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack - B/op",
            "value": 160,
            "unit": "B/op",
            "extra": "3586669 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "3586669 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf",
            "value": 300.4,
            "unit": "ns/op\t 239.71 MB/s\t     112 B/op\t       3 allocs/op",
            "extra": "8053587 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf - ns/op",
            "value": 300.4,
            "unit": "ns/op",
            "extra": "8053587 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf - MB/s",
            "value": 239.71,
            "unit": "MB/s",
            "extra": "8053587 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf - B/op",
            "value": 112,
            "unit": "B/op",
            "extra": "8053587 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "8053587 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json",
            "value": 384974,
            "unit": "ns/op\t 371.14 MB/s\t  150514 B/op\t       2 allocs/op",
            "extra": "6104 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json - ns/op",
            "value": 384974,
            "unit": "ns/op",
            "extra": "6104 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json - MB/s",
            "value": 371.14,
            "unit": "MB/s",
            "extra": "6104 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json - B/op",
            "value": 150514,
            "unit": "B/op",
            "extra": "6104 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "6104 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack",
            "value": 486311,
            "unit": "ns/op\t 229.56 MB/s\t  286202 B/op\t    1014 allocs/op",
            "extra": "4869 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack - ns/op",
            "value": 486311,
            "unit": "ns/op",
            "extra": "4869 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack - MB/s",
            "value": 229.56,
            "unit": "MB/s",
            "extra": "4869 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack - B/op",
            "value": 286202,
            "unit": "B/op",
            "extra": "4869 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack - allocs/op",
            "value": 1014,
            "unit": "allocs/op",
            "extra": "4869 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf",
            "value": 267120,
            "unit": "ns/op\t 151.85 MB/s\t   41156 B/op\t       3 allocs/op",
            "extra": "8815 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf - ns/op",
            "value": 267120,
            "unit": "ns/op",
            "extra": "8815 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf - MB/s",
            "value": 151.85,
            "unit": "MB/s",
            "extra": "8815 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf - B/op",
            "value": 41156,
            "unit": "B/op",
            "extra": "8815 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "8815 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json",
            "value": 2739137,
            "unit": "ns/op\t  52.16 MB/s\t  503579 B/op\t    9019 allocs/op",
            "extra": "878 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json - ns/op",
            "value": 2739137,
            "unit": "ns/op",
            "extra": "878 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json - MB/s",
            "value": 52.16,
            "unit": "MB/s",
            "extra": "878 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json - B/op",
            "value": 503579,
            "unit": "B/op",
            "extra": "878 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json - allocs/op",
            "value": 9019,
            "unit": "allocs/op",
            "extra": "878 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack",
            "value": 975080,
            "unit": "ns/op\t 114.49 MB/s\t  323890 B/op\t    8007 allocs/op",
            "extra": "2456 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack - ns/op",
            "value": 975080,
            "unit": "ns/op",
            "extra": "2456 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack - MB/s",
            "value": 114.49,
            "unit": "MB/s",
            "extra": "2456 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack - B/op",
            "value": 323890,
            "unit": "B/op",
            "extra": "2456 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack - allocs/op",
            "value": 8007,
            "unit": "allocs/op",
            "extra": "2456 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf",
            "value": 349547,
            "unit": "ns/op\t 116.04 MB/s\t  169203 B/op\t    3466 allocs/op",
            "extra": "6846 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf - ns/op",
            "value": 349547,
            "unit": "ns/op",
            "extra": "6846 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf - MB/s",
            "value": 116.04,
            "unit": "MB/s",
            "extra": "6846 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf - B/op",
            "value": 169203,
            "unit": "B/op",
            "extra": "6846 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf - allocs/op",
            "value": 3466,
            "unit": "allocs/op",
            "extra": "6846 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf",
            "value": 249414,
            "unit": "ns/op\t 162.63 MB/s\t      91 B/op\t       2 allocs/op",
            "extra": "9528 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf - ns/op",
            "value": 249414,
            "unit": "ns/op",
            "extra": "9528 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf - MB/s",
            "value": 162.63,
            "unit": "MB/s",
            "extra": "9528 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf - B/op",
            "value": 91,
            "unit": "B/op",
            "extra": "9528 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "9528 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json",
            "value": 153556,
            "unit": "ns/op\t 242.63 MB/s\t   41070 B/op\t       2 allocs/op",
            "extra": "15633 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json - ns/op",
            "value": 153556,
            "unit": "ns/op",
            "extra": "15633 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json - MB/s",
            "value": 242.63,
            "unit": "MB/s",
            "extra": "15633 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json - B/op",
            "value": 41070,
            "unit": "B/op",
            "extra": "15633 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "15633 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack",
            "value": 113018,
            "unit": "ns/op\t 172.64 MB/s\t   65626 B/op\t      12 allocs/op",
            "extra": "21184 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack - ns/op",
            "value": 113018,
            "unit": "ns/op",
            "extra": "21184 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack - MB/s",
            "value": 172.64,
            "unit": "MB/s",
            "extra": "21184 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack - B/op",
            "value": 65626,
            "unit": "B/op",
            "extra": "21184 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "21184 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf",
            "value": 4645,
            "unit": "ns/op\t1806.52 MB/s\t    9668 B/op\t       3 allocs/op",
            "extra": "500421 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf - ns/op",
            "value": 4645,
            "unit": "ns/op",
            "extra": "500421 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf - MB/s",
            "value": 1806.52,
            "unit": "MB/s",
            "extra": "500421 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf - B/op",
            "value": 9668,
            "unit": "B/op",
            "extra": "500421 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "500421 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json",
            "value": 567698,
            "unit": "ns/op\t  65.63 MB/s\t   54080 B/op\t      40 allocs/op",
            "extra": "4257 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json - ns/op",
            "value": 567698,
            "unit": "ns/op",
            "extra": "4257 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json - MB/s",
            "value": 65.63,
            "unit": "MB/s",
            "extra": "4257 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json - B/op",
            "value": 54080,
            "unit": "B/op",
            "extra": "4257 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json - allocs/op",
            "value": 40,
            "unit": "allocs/op",
            "extra": "4257 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack",
            "value": 147067,
            "unit": "ns/op\t 132.67 MB/s\t   35197 B/op\t      18 allocs/op",
            "extra": "16328 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack - ns/op",
            "value": 147067,
            "unit": "ns/op",
            "extra": "16328 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack - MB/s",
            "value": 132.67,
            "unit": "MB/s",
            "extra": "16328 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack - B/op",
            "value": 35197,
            "unit": "B/op",
            "extra": "16328 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "16328 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf",
            "value": 5673,
            "unit": "ns/op\t1479.11 MB/s\t   17524 B/op\t       5 allocs/op",
            "extra": "430806 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf - ns/op",
            "value": 5673,
            "unit": "ns/op",
            "extra": "430806 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf - MB/s",
            "value": 1479.11,
            "unit": "MB/s",
            "extra": "430806 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf - B/op",
            "value": 17524,
            "unit": "B/op",
            "extra": "430806 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "430806 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json",
            "value": 127165,
            "unit": "ns/op\t 228.24 MB/s\t   32893 B/op\t       2 allocs/op",
            "extra": "18826 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json - ns/op",
            "value": 127165,
            "unit": "ns/op",
            "extra": "18826 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json - MB/s",
            "value": 228.24,
            "unit": "MB/s",
            "extra": "18826 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json - B/op",
            "value": 32893,
            "unit": "B/op",
            "extra": "18826 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "18826 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack",
            "value": 112405,
            "unit": "ns/op\t 173.65 MB/s\t   65625 B/op\t      12 allocs/op",
            "extra": "21366 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack - ns/op",
            "value": 112405,
            "unit": "ns/op",
            "extra": "21366 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack - MB/s",
            "value": 173.65,
            "unit": "MB/s",
            "extra": "21366 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack - B/op",
            "value": 65625,
            "unit": "B/op",
            "extra": "21366 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "21366 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf",
            "value": 4596,
            "unit": "ns/op\t1827.18 MB/s\t    9668 B/op\t       3 allocs/op",
            "extra": "531062 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf - ns/op",
            "value": 4596,
            "unit": "ns/op",
            "extra": "531062 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf - MB/s",
            "value": 1827.18,
            "unit": "MB/s",
            "extra": "531062 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf - B/op",
            "value": 9668,
            "unit": "B/op",
            "extra": "531062 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "531062 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json",
            "value": 482143,
            "unit": "ns/op\t  60.20 MB/s\t   54088 B/op\t      40 allocs/op",
            "extra": "5002 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json - ns/op",
            "value": 482143,
            "unit": "ns/op",
            "extra": "5002 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json - MB/s",
            "value": 60.2,
            "unit": "MB/s",
            "extra": "5002 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json - B/op",
            "value": 54088,
            "unit": "B/op",
            "extra": "5002 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json - allocs/op",
            "value": 40,
            "unit": "allocs/op",
            "extra": "5002 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack",
            "value": 146858,
            "unit": "ns/op\t 132.91 MB/s\t   35205 B/op\t      18 allocs/op",
            "extra": "16324 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack - ns/op",
            "value": 146858,
            "unit": "ns/op",
            "extra": "16324 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack - MB/s",
            "value": 132.91,
            "unit": "MB/s",
            "extra": "16324 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack - B/op",
            "value": 35205,
            "unit": "B/op",
            "extra": "16324 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "16324 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf",
            "value": 5511,
            "unit": "ns/op\t1523.86 MB/s\t   17533 B/op\t       5 allocs/op",
            "extra": "417990 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf - ns/op",
            "value": 5511,
            "unit": "ns/op",
            "extra": "417990 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf - MB/s",
            "value": 1523.86,
            "unit": "MB/s",
            "extra": "417990 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf - B/op",
            "value": 17533,
            "unit": "B/op",
            "extra": "417990 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "417990 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json",
            "value": 127952,
            "unit": "ns/op\t 226.84 MB/s\t   32875 B/op\t       2 allocs/op",
            "extra": "18771 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json - ns/op",
            "value": 127952,
            "unit": "ns/op",
            "extra": "18771 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json - MB/s",
            "value": 226.84,
            "unit": "MB/s",
            "extra": "18771 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json - B/op",
            "value": 32875,
            "unit": "B/op",
            "extra": "18771 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "18771 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack",
            "value": 112853,
            "unit": "ns/op\t 172.96 MB/s\t   65626 B/op\t      12 allocs/op",
            "extra": "21178 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack - ns/op",
            "value": 112853,
            "unit": "ns/op",
            "extra": "21178 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack - MB/s",
            "value": 172.96,
            "unit": "MB/s",
            "extra": "21178 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack - B/op",
            "value": 65626,
            "unit": "B/op",
            "extra": "21178 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "21178 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf",
            "value": 51560,
            "unit": "ns/op\t  44.74 MB/s\t   11324 B/op\t      14 allocs/op",
            "extra": "46537 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf - ns/op",
            "value": 51560,
            "unit": "ns/op",
            "extra": "46537 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf - MB/s",
            "value": 44.74,
            "unit": "MB/s",
            "extra": "46537 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf - B/op",
            "value": 11324,
            "unit": "B/op",
            "extra": "46537 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf - allocs/op",
            "value": 14,
            "unit": "allocs/op",
            "extra": "46537 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json",
            "value": 484323,
            "unit": "ns/op\t  59.93 MB/s\t   54088 B/op\t      40 allocs/op",
            "extra": "4921 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json - ns/op",
            "value": 484323,
            "unit": "ns/op",
            "extra": "4921 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json - MB/s",
            "value": 59.93,
            "unit": "MB/s",
            "extra": "4921 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json - B/op",
            "value": 54088,
            "unit": "B/op",
            "extra": "4921 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json - allocs/op",
            "value": 40,
            "unit": "allocs/op",
            "extra": "4921 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack",
            "value": 148156,
            "unit": "ns/op\t 131.75 MB/s\t   35205 B/op\t      18 allocs/op",
            "extra": "16131 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack - ns/op",
            "value": 148156,
            "unit": "ns/op",
            "extra": "16131 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack - MB/s",
            "value": 131.75,
            "unit": "MB/s",
            "extra": "16131 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack - B/op",
            "value": 35205,
            "unit": "B/op",
            "extra": "16131 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "16131 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf",
            "value": 47020,
            "unit": "ns/op\t  49.06 MB/s\t   17612 B/op\t       7 allocs/op",
            "extra": "50996 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf - ns/op",
            "value": 47020,
            "unit": "ns/op",
            "extra": "50996 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf - MB/s",
            "value": 49.06,
            "unit": "MB/s",
            "extra": "50996 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf - B/op",
            "value": 17612,
            "unit": "B/op",
            "extra": "50996 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "50996 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json",
            "value": 24338,
            "unit": "ns/op\t 169.94 MB/s\t    4913 B/op\t       2 allocs/op",
            "extra": "99560 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json - ns/op",
            "value": 24338,
            "unit": "ns/op",
            "extra": "99560 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json - MB/s",
            "value": 169.94,
            "unit": "MB/s",
            "extra": "99560 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json - B/op",
            "value": 4913,
            "unit": "B/op",
            "extra": "99560 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "99560 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack",
            "value": 45926,
            "unit": "ns/op\t 201.43 MB/s\t   32804 B/op\t      11 allocs/op",
            "extra": "51782 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack - ns/op",
            "value": 45926,
            "unit": "ns/op",
            "extra": "51782 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack - MB/s",
            "value": 201.43,
            "unit": "MB/s",
            "extra": "51782 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack - B/op",
            "value": 32804,
            "unit": "B/op",
            "extra": "51782 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "51782 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf",
            "value": 4938,
            "unit": "ns/op\t  20.05 MB/s\t     208 B/op\t       3 allocs/op",
            "extra": "492162 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf - ns/op",
            "value": 4938,
            "unit": "ns/op",
            "extra": "492162 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf - MB/s",
            "value": 20.05,
            "unit": "MB/s",
            "extra": "492162 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf - B/op",
            "value": 208,
            "unit": "B/op",
            "extra": "492162 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "492162 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json",
            "value": 129371,
            "unit": "ns/op\t  31.97 MB/s\t   25504 B/op\t      19 allocs/op",
            "extra": "18591 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json - ns/op",
            "value": 129371,
            "unit": "ns/op",
            "extra": "18591 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json - MB/s",
            "value": 31.97,
            "unit": "MB/s",
            "extra": "18591 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json - B/op",
            "value": 25504,
            "unit": "B/op",
            "extra": "18591 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json - allocs/op",
            "value": 19,
            "unit": "allocs/op",
            "extra": "18591 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack",
            "value": 58079,
            "unit": "ns/op\t 159.28 MB/s\t   16570 B/op\t       8 allocs/op",
            "extra": "41239 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack - ns/op",
            "value": 58079,
            "unit": "ns/op",
            "extra": "41239 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack - MB/s",
            "value": 159.28,
            "unit": "MB/s",
            "extra": "41239 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack - B/op",
            "value": 16570,
            "unit": "B/op",
            "extra": "41239 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "41239 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf",
            "value": 2810,
            "unit": "ns/op\t  35.23 MB/s\t    8258 B/op\t       3 allocs/op",
            "extra": "853761 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf - ns/op",
            "value": 2810,
            "unit": "ns/op",
            "extra": "853761 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf - MB/s",
            "value": 35.23,
            "unit": "MB/s",
            "extra": "853761 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf - B/op",
            "value": 8258,
            "unit": "B/op",
            "extra": "853761 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "853761 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json",
            "value": 59996,
            "unit": "ns/op\t 139.76 MB/s\t    9522 B/op\t       2 allocs/op",
            "extra": "39922 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json - ns/op",
            "value": 59996,
            "unit": "ns/op",
            "extra": "39922 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json - MB/s",
            "value": 139.76,
            "unit": "MB/s",
            "extra": "39922 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json - B/op",
            "value": 9522,
            "unit": "B/op",
            "extra": "39922 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "39922 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack",
            "value": 28595,
            "unit": "ns/op\t 135.13 MB/s\t    8225 B/op\t       9 allocs/op",
            "extra": "84332 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack - ns/op",
            "value": 28595,
            "unit": "ns/op",
            "extra": "84332 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack - MB/s",
            "value": 135.13,
            "unit": "MB/s",
            "extra": "84332 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack - B/op",
            "value": 8225,
            "unit": "B/op",
            "extra": "84332 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack - allocs/op",
            "value": 9,
            "unit": "allocs/op",
            "extra": "84332 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf",
            "value": 1052,
            "unit": "ns/op\t2950.75 MB/s\t    3297 B/op\t       3 allocs/op",
            "extra": "2259994 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf - ns/op",
            "value": 1052,
            "unit": "ns/op",
            "extra": "2259994 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf - MB/s",
            "value": 2950.75,
            "unit": "MB/s",
            "extra": "2259994 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf - B/op",
            "value": 3297,
            "unit": "B/op",
            "extra": "2259994 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "2259994 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json",
            "value": 165405,
            "unit": "ns/op\t  50.69 MB/s\t    7832 B/op\t      17 allocs/op",
            "extra": "14499 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json - ns/op",
            "value": 165405,
            "unit": "ns/op",
            "extra": "14499 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json - MB/s",
            "value": 50.69,
            "unit": "MB/s",
            "extra": "14499 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json - B/op",
            "value": 7832,
            "unit": "B/op",
            "extra": "14499 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "14499 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack",
            "value": 40748,
            "unit": "ns/op\t  94.83 MB/s\t    6320 B/op\t       8 allocs/op",
            "extra": "59031 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack - ns/op",
            "value": 40748,
            "unit": "ns/op",
            "extra": "59031 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack - MB/s",
            "value": 94.83,
            "unit": "MB/s",
            "extra": "59031 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack - B/op",
            "value": 6320,
            "unit": "B/op",
            "extra": "59031 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "59031 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf",
            "value": 866.4,
            "unit": "ns/op\t3581.47 MB/s\t    3129 B/op\t       3 allocs/op",
            "extra": "2743687 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf - ns/op",
            "value": 866.4,
            "unit": "ns/op",
            "extra": "2743687 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf - MB/s",
            "value": 3581.47,
            "unit": "MB/s",
            "extra": "2743687 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf - B/op",
            "value": 3129,
            "unit": "B/op",
            "extra": "2743687 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "2743687 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json",
            "value": 2106,
            "unit": "ns/op\t 118.74 MB/s\t     936 B/op\t      22 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json - ns/op",
            "value": 2106,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json - MB/s",
            "value": 118.74,
            "unit": "MB/s",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json - B/op",
            "value": 936,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json - allocs/op",
            "value": 22,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack",
            "value": 1682,
            "unit": "ns/op\t 117.12 MB/s\t     680 B/op\t      15 allocs/op",
            "extra": "1431469 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack - ns/op",
            "value": 1682,
            "unit": "ns/op",
            "extra": "1431469 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack - MB/s",
            "value": 117.12,
            "unit": "MB/s",
            "extra": "1431469 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack - B/op",
            "value": 680,
            "unit": "B/op",
            "extra": "1431469 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "1431469 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf",
            "value": 1331,
            "unit": "ns/op\t 169.07 MB/s\t     368 B/op\t       3 allocs/op",
            "extra": "1801683 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf - ns/op",
            "value": 1331,
            "unit": "ns/op",
            "extra": "1801683 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf - MB/s",
            "value": 169.07,
            "unit": "MB/s",
            "extra": "1801683 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf - B/op",
            "value": 368,
            "unit": "B/op",
            "extra": "1801683 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1801683 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json",
            "value": 6069,
            "unit": "ns/op\t  41.19 MB/s\t    1352 B/op\t      41 allocs/op",
            "extra": "400198 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json - ns/op",
            "value": 6069,
            "unit": "ns/op",
            "extra": "400198 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json - MB/s",
            "value": 41.19,
            "unit": "MB/s",
            "extra": "400198 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json - B/op",
            "value": 1352,
            "unit": "B/op",
            "extra": "400198 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json - allocs/op",
            "value": 41,
            "unit": "allocs/op",
            "extra": "400198 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack",
            "value": 2753,
            "unit": "ns/op\t  71.56 MB/s\t    1064 B/op\t      34 allocs/op",
            "extra": "875082 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack - ns/op",
            "value": 2753,
            "unit": "ns/op",
            "extra": "875082 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack - MB/s",
            "value": 71.56,
            "unit": "MB/s",
            "extra": "875082 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack - B/op",
            "value": 1064,
            "unit": "B/op",
            "extra": "875082 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack - allocs/op",
            "value": 34,
            "unit": "allocs/op",
            "extra": "875082 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf",
            "value": 1631,
            "unit": "ns/op\t 137.95 MB/s\t     889 B/op\t      16 allocs/op",
            "extra": "1463737 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf - ns/op",
            "value": 1631,
            "unit": "ns/op",
            "extra": "1463737 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf - MB/s",
            "value": 137.95,
            "unit": "MB/s",
            "extra": "1463737 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf - B/op",
            "value": 889,
            "unit": "B/op",
            "extra": "1463737 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf - allocs/op",
            "value": 16,
            "unit": "allocs/op",
            "extra": "1463737 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json",
            "value": 1866415,
            "unit": "ns/op\t 382.98 MB/s\t  727902 B/op\t       3 allocs/op",
            "extra": "1245 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json - ns/op",
            "value": 1866415,
            "unit": "ns/op",
            "extra": "1245 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json - MB/s",
            "value": 382.98,
            "unit": "MB/s",
            "extra": "1245 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json - B/op",
            "value": 727902,
            "unit": "B/op",
            "extra": "1245 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1245 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack",
            "value": 2558570,
            "unit": "ns/op\t 218.29 MB/s\t 2217444 B/op\t    5018 allocs/op",
            "extra": "937 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack - ns/op",
            "value": 2558570,
            "unit": "ns/op",
            "extra": "937 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack - MB/s",
            "value": 218.29,
            "unit": "MB/s",
            "extra": "937 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack - B/op",
            "value": 2217444,
            "unit": "B/op",
            "extra": "937 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack - allocs/op",
            "value": 5018,
            "unit": "allocs/op",
            "extra": "937 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf",
            "value": 1921651,
            "unit": "ns/op\t 100.04 MB/s\t 1518642 B/op\t      51 allocs/op",
            "extra": "1237 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf - ns/op",
            "value": 1921651,
            "unit": "ns/op",
            "extra": "1237 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf - MB/s",
            "value": 100.04,
            "unit": "MB/s",
            "extra": "1237 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf - B/op",
            "value": 1518642,
            "unit": "B/op",
            "extra": "1237 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "1237 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json",
            "value": 13753968,
            "unit": "ns/op\t  51.97 MB/s\t 3074431 B/op\t   45025 allocs/op",
            "extra": "174 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json - ns/op",
            "value": 13753968,
            "unit": "ns/op",
            "extra": "174 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json - MB/s",
            "value": 51.97,
            "unit": "MB/s",
            "extra": "174 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json - B/op",
            "value": 3074431,
            "unit": "B/op",
            "extra": "174 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json - allocs/op",
            "value": 45025,
            "unit": "allocs/op",
            "extra": "174 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack",
            "value": 5050330,
            "unit": "ns/op\t 110.59 MB/s\t 1602208 B/op\t   40008 allocs/op",
            "extra": "474 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack - ns/op",
            "value": 5050330,
            "unit": "ns/op",
            "extra": "474 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack - MB/s",
            "value": 110.59,
            "unit": "MB/s",
            "extra": "474 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack - B/op",
            "value": 1602208,
            "unit": "B/op",
            "extra": "474 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack - allocs/op",
            "value": 40008,
            "unit": "allocs/op",
            "extra": "474 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf",
            "value": 2417535,
            "unit": "ns/op\t  79.52 MB/s\t 1822988 B/op\t   16295 allocs/op",
            "extra": "976 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf - ns/op",
            "value": 2417535,
            "unit": "ns/op",
            "extra": "976 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf - MB/s",
            "value": 79.52,
            "unit": "MB/s",
            "extra": "976 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf - B/op",
            "value": 1822988,
            "unit": "B/op",
            "extra": "976 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf - allocs/op",
            "value": 16295,
            "unit": "allocs/op",
            "extra": "976 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/json",
            "value": 47552,
            "unit": "ns/op\t   10980 B/op\t       2 allocs/op",
            "extra": "50612 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/json - ns/op",
            "value": 47552,
            "unit": "ns/op",
            "extra": "50612 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/json - B/op",
            "value": 10980,
            "unit": "B/op",
            "extra": "50612 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "50612 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/msgpack",
            "value": 59826,
            "unit": "ns/op\t   32852 B/op\t      11 allocs/op",
            "extra": "39153 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/msgpack - ns/op",
            "value": 59826,
            "unit": "ns/op",
            "extra": "39153 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/msgpack - B/op",
            "value": 32852,
            "unit": "B/op",
            "extra": "39153 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/msgpack - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "39153 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_fast",
            "value": 7975,
            "unit": "ns/op\t    6978 B/op\t       3 allocs/op",
            "extra": "295020 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_fast - ns/op",
            "value": 7975,
            "unit": "ns/op",
            "extra": "295020 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_fast - B/op",
            "value": 6978,
            "unit": "B/op",
            "extra": "295020 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "295020 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_qpack",
            "value": 2994,
            "unit": "ns/op\t    2496 B/op\t       3 allocs/op",
            "extra": "752505 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_qpack - ns/op",
            "value": 2994,
            "unit": "ns/op",
            "extra": "752505 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_qpack - B/op",
            "value": 2496,
            "unit": "B/op",
            "extra": "752505 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_qpack - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "752505 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_dense",
            "value": 3141,
            "unit": "ns/op\t    2497 B/op\t       3 allocs/op",
            "extra": "761460 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_dense - ns/op",
            "value": 3141,
            "unit": "ns/op",
            "extra": "761460 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_dense - B/op",
            "value": 2497,
            "unit": "B/op",
            "extra": "761460 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "761460 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/json",
            "value": 215830,
            "unit": "ns/op\t   21288 B/op\t      41 allocs/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/json - ns/op",
            "value": 215830,
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
            "value": 83796,
            "unit": "ns/op\t   21427 B/op\t      22 allocs/op",
            "extra": "28719 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/msgpack - ns/op",
            "value": 83796,
            "unit": "ns/op",
            "extra": "28719 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/msgpack - B/op",
            "value": 21427,
            "unit": "B/op",
            "extra": "28719 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/msgpack - allocs/op",
            "value": 22,
            "unit": "allocs/op",
            "extra": "28719 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_fast",
            "value": 15153,
            "unit": "ns/op\t   10594 B/op\t       5 allocs/op",
            "extra": "157730 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_fast - ns/op",
            "value": 15153,
            "unit": "ns/op",
            "extra": "157730 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_fast - B/op",
            "value": 10594,
            "unit": "B/op",
            "extra": "157730 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_fast - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "157730 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_qpack",
            "value": 3518,
            "unit": "ns/op\t   10595 B/op\t       5 allocs/op",
            "extra": "679083 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_qpack - ns/op",
            "value": 3518,
            "unit": "ns/op",
            "extra": "679083 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_qpack - B/op",
            "value": 10595,
            "unit": "B/op",
            "extra": "679083 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_qpack - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "679083 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_dense",
            "value": 3725,
            "unit": "ns/op\t   10676 B/op\t       7 allocs/op",
            "extra": "625564 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_dense - ns/op",
            "value": 3725,
            "unit": "ns/op",
            "extra": "625564 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_dense - B/op",
            "value": 10676,
            "unit": "B/op",
            "extra": "625564 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_dense - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "625564 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/json",
            "value": 3827,
            "unit": "ns/op\t     488 B/op\t      12 allocs/op",
            "extra": "639246 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/json - ns/op",
            "value": 3827,
            "unit": "ns/op",
            "extra": "639246 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/json - B/op",
            "value": 488,
            "unit": "B/op",
            "extra": "639246 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/json - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "639246 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/msgpack",
            "value": 1235,
            "unit": "ns/op\t     312 B/op\t       9 allocs/op",
            "extra": "1944511 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/msgpack - ns/op",
            "value": 1235,
            "unit": "ns/op",
            "extra": "1944511 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/msgpack - B/op",
            "value": 312,
            "unit": "B/op",
            "extra": "1944511 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/msgpack - allocs/op",
            "value": 9,
            "unit": "allocs/op",
            "extra": "1944511 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/qdf_fast",
            "value": 576.3,
            "unit": "ns/op\t     264 B/op\t       8 allocs/op",
            "extra": "4166262 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/qdf_fast - ns/op",
            "value": 576.3,
            "unit": "ns/op",
            "extra": "4166262 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/qdf_fast - B/op",
            "value": 264,
            "unit": "B/op",
            "extra": "4166262 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/qdf_fast - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "4166262 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/json",
            "value": 1654,
            "unit": "ns/op\t     488 B/op\t      12 allocs/op",
            "extra": "1456676 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/json - ns/op",
            "value": 1654,
            "unit": "ns/op",
            "extra": "1456676 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/json - B/op",
            "value": 488,
            "unit": "B/op",
            "extra": "1456676 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/json - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "1456676 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/msgpack",
            "value": 562.1,
            "unit": "ns/op\t     312 B/op\t       9 allocs/op",
            "extra": "4292804 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/msgpack - ns/op",
            "value": 562.1,
            "unit": "ns/op",
            "extra": "4292804 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/msgpack - B/op",
            "value": 312,
            "unit": "B/op",
            "extra": "4292804 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/msgpack - allocs/op",
            "value": 9,
            "unit": "allocs/op",
            "extra": "4292804 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/qdf_fast",
            "value": 275.9,
            "unit": "ns/op\t     264 B/op\t       8 allocs/op",
            "extra": "8615253 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/qdf_fast - ns/op",
            "value": 275.9,
            "unit": "ns/op",
            "extra": "8615253 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/qdf_fast - B/op",
            "value": 264,
            "unit": "B/op",
            "extra": "8615253 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/qdf_fast - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "8615253 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense",
            "value": 411236,
            "unit": "ns/op\t 451.54 MB/s\t  376005 B/op\t    2011 allocs/op",
            "extra": "5802 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense - ns/op",
            "value": 411236,
            "unit": "ns/op",
            "extra": "5802 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense - MB/s",
            "value": 451.54,
            "unit": "MB/s",
            "extra": "5802 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense - B/op",
            "value": 376005,
            "unit": "B/op",
            "extra": "5802 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense - allocs/op",
            "value": 2011,
            "unit": "allocs/op",
            "extra": "5802 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/json",
            "value": 2378,
            "unit": "ns/op\t     800 B/op\t      14 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/json - ns/op",
            "value": 2378,
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
            "value": 2032,
            "unit": "ns/op\t    1008 B/op\t      17 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/msgpack - ns/op",
            "value": 2032,
            "unit": "ns/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/msgpack - B/op",
            "value": 1008,
            "unit": "B/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/msgpack - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/qdf_fast",
            "value": 1679,
            "unit": "ns/op\t     696 B/op\t      13 allocs/op",
            "extra": "1429708 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/qdf_fast - ns/op",
            "value": 1679,
            "unit": "ns/op",
            "extra": "1429708 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/qdf_fast - B/op",
            "value": 696,
            "unit": "B/op",
            "extra": "1429708 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/qdf_fast - allocs/op",
            "value": 13,
            "unit": "allocs/op",
            "extra": "1429708 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/json",
            "value": 1101,
            "unit": "ns/op\t     364 B/op\t       5 allocs/op",
            "extra": "2166015 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/json - ns/op",
            "value": 1101,
            "unit": "ns/op",
            "extra": "2166015 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/json - B/op",
            "value": 364,
            "unit": "B/op",
            "extra": "2166015 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/json - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "2166015 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/msgpack",
            "value": 1087,
            "unit": "ns/op\t     538 B/op\t       7 allocs/op",
            "extra": "2213599 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/msgpack - ns/op",
            "value": 1087,
            "unit": "ns/op",
            "extra": "2213599 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/msgpack - B/op",
            "value": 538,
            "unit": "B/op",
            "extra": "2213599 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/msgpack - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "2213599 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/qdf_fast",
            "value": 836.8,
            "unit": "ns/op\t     388 B/op\t       5 allocs/op",
            "extra": "2859945 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/qdf_fast - ns/op",
            "value": 836.8,
            "unit": "ns/op",
            "extra": "2859945 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/qdf_fast - B/op",
            "value": 388,
            "unit": "B/op",
            "extra": "2859945 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/qdf_fast - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "2859945 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/json",
            "value": 299054,
            "unit": "ns/op\t  122644 B/op\t       3 allocs/op",
            "extra": "7670 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/json - ns/op",
            "value": 299054,
            "unit": "ns/op",
            "extra": "7670 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/json - B/op",
            "value": 122644,
            "unit": "B/op",
            "extra": "7670 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/json - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "7670 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/msgpack",
            "value": 251395,
            "unit": "ns/op\t  190328 B/op\t      10 allocs/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/msgpack - ns/op",
            "value": 251395,
            "unit": "ns/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/msgpack - B/op",
            "value": 190328,
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
            "value": 94618,
            "unit": "ns/op\t   91065 B/op\t       4 allocs/op",
            "extra": "25413 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/qdf_fast - ns/op",
            "value": 94618,
            "unit": "ns/op",
            "extra": "25413 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/qdf_fast - B/op",
            "value": 91065,
            "unit": "B/op",
            "extra": "25413 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/qdf_fast - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "25413 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/json",
            "value": 1106,
            "unit": "ns/op\t     800 B/op\t      14 allocs/op",
            "extra": "2173812 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/json - ns/op",
            "value": 1106,
            "unit": "ns/op",
            "extra": "2173812 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/json - B/op",
            "value": 800,
            "unit": "B/op",
            "extra": "2173812 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/json - allocs/op",
            "value": 14,
            "unit": "allocs/op",
            "extra": "2173812 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/msgpack",
            "value": 1004,
            "unit": "ns/op\t    1008 B/op\t      17 allocs/op",
            "extra": "2360136 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/msgpack - ns/op",
            "value": 1004,
            "unit": "ns/op",
            "extra": "2360136 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/msgpack - B/op",
            "value": 1008,
            "unit": "B/op",
            "extra": "2360136 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/msgpack - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "2360136 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/qdf_fast",
            "value": 799.5,
            "unit": "ns/op\t     697 B/op\t      13 allocs/op",
            "extra": "3006498 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/qdf_fast - ns/op",
            "value": 799.5,
            "unit": "ns/op",
            "extra": "3006498 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/qdf_fast - B/op",
            "value": 697,
            "unit": "B/op",
            "extra": "3006498 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/qdf_fast - allocs/op",
            "value": 13,
            "unit": "allocs/op",
            "extra": "3006498 times\n4 procs"
          }
        ]
      },
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
          "id": "7b485303d1948d0878d348b124aec526854bafc2",
          "message": "qpack: AVX2 VPSRLVQ decode for wide FOR widths 15-28\n\nExtends SIMD FOR decode past the 4-values-per-window range. For 15<=b<=28\nfour values overflow a 64-bit window, but two fit even at the worst byte\noffset (7 + 2*28 = 63 < 64), so a 2-value/iter kernel handles them:\nVPBROADCASTQ the 8-byte window to both lanes, VPSRLVQ by a per-pair shift\n[off, off+b] from an offset-indexed table, AND mask. bitUnpackU64LE routes\n15<=b<=28 (those not already special-cased) through it; widths 29+ stay\nscalar. Byte-aligned handoff and 8-byte read headroom bound the group count.\n\nOutput bit-identical to scalar (parity across widths 15/17/18/19/21/23/24/28\nand many sizes under both build configs; golden unchanged). Opt-in qdf_simd.\n\nMicrobench (1024 elems, -count=5, i7-9750H), scalar -> SIMD:\n  width 17: ~3650 -> ~750 ns  (~4.9x)\n  width 23: ~3900 -> ~750 ns  (~5.2x)\n  width 28: ~4760 -> ~690 ns  (~6.9x)\n\nWith this, every FOR width 1-28 plus 32 has a SIMD decode path. Docs\n(USAGE/GUIDE/README) updated.",
          "timestamp": "2026-05-29T14:28:00+03:00",
          "tree_id": "4cad71983862046566ce5bdb20621ed3506c1bc9",
          "url": "https://github.com/alex60217101990/qdf/commit/7b485303d1948d0878d348b124aec526854bafc2"
        },
        "date": 1780054580936,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json",
            "value": 167.8,
            "unit": "ns/op\t 148.98 MB/s\t      24 B/op\t       1 allocs/op",
            "extra": "14640728 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json - ns/op",
            "value": 167.8,
            "unit": "ns/op",
            "extra": "14640728 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json - MB/s",
            "value": 148.98,
            "unit": "MB/s",
            "extra": "14640728 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json - B/op",
            "value": 24,
            "unit": "B/op",
            "extra": "14640728 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "14640728 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal",
            "value": 190,
            "unit": "ns/op\t 126.33 MB/s\t      48 B/op\t       2 allocs/op",
            "extra": "12550542 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal - ns/op",
            "value": 190,
            "unit": "ns/op",
            "extra": "12550542 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal - MB/s",
            "value": 126.33,
            "unit": "MB/s",
            "extra": "12550542 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal - B/op",
            "value": 48,
            "unit": "B/op",
            "extra": "12550542 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/json_marshal - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "12550542 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack",
            "value": 248.3,
            "unit": "ns/op\t  64.45 MB/s\t     136 B/op\t       3 allocs/op",
            "extra": "9727794 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack - ns/op",
            "value": 248.3,
            "unit": "ns/op",
            "extra": "9727794 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack - MB/s",
            "value": 64.45,
            "unit": "MB/s",
            "extra": "9727794 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack - B/op",
            "value": 136,
            "unit": "B/op",
            "extra": "9727794 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/msgpack - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "9727794 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast",
            "value": 320.2,
            "unit": "ns/op\t  68.71 MB/s\t      72 B/op\t       3 allocs/op",
            "extra": "7593925 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast - ns/op",
            "value": 320.2,
            "unit": "ns/op",
            "extra": "7593925 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast - MB/s",
            "value": 68.71,
            "unit": "MB/s",
            "extra": "7593925 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast - B/op",
            "value": 72,
            "unit": "B/op",
            "extra": "7593925 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "7593925 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense",
            "value": 392.8,
            "unit": "ns/op\t  63.64 MB/s\t      80 B/op\t       3 allocs/op",
            "extra": "6166612 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense - ns/op",
            "value": 392.8,
            "unit": "ns/op",
            "extra": "6166612 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense - MB/s",
            "value": 63.64,
            "unit": "MB/s",
            "extra": "6166612 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "6166612 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Tiny/Tiny/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "6166612 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json",
            "value": 1018,
            "unit": "ns/op\t 207.31 MB/s\t     192 B/op\t       1 allocs/op",
            "extra": "2349505 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json - ns/op",
            "value": 1018,
            "unit": "ns/op",
            "extra": "2349505 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json - MB/s",
            "value": 207.31,
            "unit": "MB/s",
            "extra": "2349505 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json - B/op",
            "value": 192,
            "unit": "B/op",
            "extra": "2349505 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2349505 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal",
            "value": 1063,
            "unit": "ns/op\t 197.57 MB/s\t     416 B/op\t       2 allocs/op",
            "extra": "2262849 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal - ns/op",
            "value": 1063,
            "unit": "ns/op",
            "extra": "2262849 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal - MB/s",
            "value": 197.57,
            "unit": "MB/s",
            "extra": "2262849 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal - B/op",
            "value": 416,
            "unit": "B/op",
            "extra": "2262849 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/json_marshal - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "2262849 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack",
            "value": 1022,
            "unit": "ns/op\t 131.16 MB/s\t     688 B/op\t       5 allocs/op",
            "extra": "2348589 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack - ns/op",
            "value": 1022,
            "unit": "ns/op",
            "extra": "2348589 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack - MB/s",
            "value": 131.16,
            "unit": "MB/s",
            "extra": "2348589 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack - B/op",
            "value": 688,
            "unit": "B/op",
            "extra": "2348589 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/msgpack - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "2348589 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast",
            "value": 599.7,
            "unit": "ns/op\t 220.09 MB/s\t     528 B/op\t       3 allocs/op",
            "extra": "3992455 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast - ns/op",
            "value": 599.7,
            "unit": "ns/op",
            "extra": "3992455 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast - MB/s",
            "value": 220.09,
            "unit": "MB/s",
            "extra": "3992455 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast - B/op",
            "value": 528,
            "unit": "B/op",
            "extra": "3992455 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3992455 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense",
            "value": 769.2,
            "unit": "ns/op\t 179.40 MB/s\t     528 B/op\t       3 allocs/op",
            "extra": "3111441 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense - ns/op",
            "value": 769.2,
            "unit": "ns/op",
            "extra": "3111441 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense - MB/s",
            "value": 179.4,
            "unit": "MB/s",
            "extra": "3111441 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense - B/op",
            "value": 528,
            "unit": "B/op",
            "extra": "3111441 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Flat/Flat/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3111441 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json",
            "value": 456.8,
            "unit": "ns/op\t 227.67 MB/s\t      80 B/op\t       1 allocs/op",
            "extra": "5244250 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json - ns/op",
            "value": 456.8,
            "unit": "ns/op",
            "extra": "5244250 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json - MB/s",
            "value": 227.67,
            "unit": "MB/s",
            "extra": "5244250 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json - B/op",
            "value": 80,
            "unit": "B/op",
            "extra": "5244250 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "5244250 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal",
            "value": 489.7,
            "unit": "ns/op\t 210.34 MB/s\t     192 B/op\t       2 allocs/op",
            "extra": "4843768 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal - ns/op",
            "value": 489.7,
            "unit": "ns/op",
            "extra": "4843768 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal - MB/s",
            "value": 210.34,
            "unit": "MB/s",
            "extra": "4843768 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal - B/op",
            "value": 192,
            "unit": "B/op",
            "extra": "4843768 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/json_marshal - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "4843768 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack",
            "value": 746.4,
            "unit": "ns/op\t 101.83 MB/s\t     320 B/op\t       4 allocs/op",
            "extra": "3236095 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack - ns/op",
            "value": 746.4,
            "unit": "ns/op",
            "extra": "3236095 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack - MB/s",
            "value": 101.83,
            "unit": "MB/s",
            "extra": "3236095 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack - B/op",
            "value": 320,
            "unit": "B/op",
            "extra": "3236095 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/msgpack - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "3236095 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast",
            "value": 450.4,
            "unit": "ns/op\t 190.95 MB/s\t     256 B/op\t       3 allocs/op",
            "extra": "5303112 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast - ns/op",
            "value": 450.4,
            "unit": "ns/op",
            "extra": "5303112 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast - MB/s",
            "value": 190.95,
            "unit": "MB/s",
            "extra": "5303112 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast - B/op",
            "value": 256,
            "unit": "B/op",
            "extra": "5303112 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "5303112 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense",
            "value": 645.7,
            "unit": "ns/op\t 148.68 MB/s\t     256 B/op\t       3 allocs/op",
            "extra": "3683222 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense - ns/op",
            "value": 645.7,
            "unit": "ns/op",
            "extra": "3683222 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense - MB/s",
            "value": 148.68,
            "unit": "MB/s",
            "extra": "3683222 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense - B/op",
            "value": 256,
            "unit": "B/op",
            "extra": "3683222 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Nested/Nested/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3683222 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json",
            "value": 1532,
            "unit": "ns/op\t 156.67 MB/s\t       0 B/op\t       0 allocs/op",
            "extra": "1566952 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json - ns/op",
            "value": 1532,
            "unit": "ns/op",
            "extra": "1566952 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json - MB/s",
            "value": 156.67,
            "unit": "MB/s",
            "extra": "1566952 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json - B/op",
            "value": 0,
            "unit": "B/op",
            "extra": "1566952 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json - allocs/op",
            "value": 0,
            "unit": "allocs/op",
            "extra": "1566952 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal",
            "value": 1590,
            "unit": "ns/op\t 150.28 MB/s\t     240 B/op\t       1 allocs/op",
            "extra": "1510567 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal - ns/op",
            "value": 1590,
            "unit": "ns/op",
            "extra": "1510567 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal - MB/s",
            "value": 150.28,
            "unit": "MB/s",
            "extra": "1510567 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal - B/op",
            "value": 240,
            "unit": "B/op",
            "extra": "1510567 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/json_marshal - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "1510567 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack",
            "value": 2915,
            "unit": "ns/op\t  47.69 MB/s\t     752 B/op\t      20 allocs/op",
            "extra": "809194 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack - ns/op",
            "value": 2915,
            "unit": "ns/op",
            "extra": "809194 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack - MB/s",
            "value": 47.69,
            "unit": "MB/s",
            "extra": "809194 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack - B/op",
            "value": 752,
            "unit": "B/op",
            "extra": "809194 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/msgpack - allocs/op",
            "value": 20,
            "unit": "allocs/op",
            "extra": "809194 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast",
            "value": 755,
            "unit": "ns/op\t 219.88 MB/s\t     176 B/op\t       1 allocs/op",
            "extra": "3184611 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast - ns/op",
            "value": 755,
            "unit": "ns/op",
            "extra": "3184611 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast - MB/s",
            "value": 219.88,
            "unit": "MB/s",
            "extra": "3184611 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast - B/op",
            "value": 176,
            "unit": "B/op",
            "extra": "3184611 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_fast - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "3184611 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense",
            "value": 706.4,
            "unit": "ns/op\t  89.18 MB/s\t      64 B/op\t       1 allocs/op",
            "extra": "3384014 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense - ns/op",
            "value": 706.4,
            "unit": "ns/op",
            "extra": "3384014 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense - MB/s",
            "value": 89.18,
            "unit": "MB/s",
            "extra": "3384014 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense - B/op",
            "value": 64,
            "unit": "B/op",
            "extra": "3384014 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Deep16/Deep16/qdf_dense - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "3384014 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json",
            "value": 875543,
            "unit": "ns/op\t 243.17 MB/s\t     290 B/op\t       1 allocs/op",
            "extra": "2772 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json - ns/op",
            "value": 875543,
            "unit": "ns/op",
            "extra": "2772 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json - MB/s",
            "value": 243.17,
            "unit": "MB/s",
            "extra": "2772 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json - B/op",
            "value": 290,
            "unit": "B/op",
            "extra": "2772 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json - allocs/op",
            "value": 1,
            "unit": "allocs/op",
            "extra": "2772 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal",
            "value": 894316,
            "unit": "ns/op\t 238.06 MB/s\t  213654 B/op\t       2 allocs/op",
            "extra": "2599 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal - ns/op",
            "value": 894316,
            "unit": "ns/op",
            "extra": "2599 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal - MB/s",
            "value": 238.06,
            "unit": "MB/s",
            "extra": "2599 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal - B/op",
            "value": 213654,
            "unit": "B/op",
            "extra": "2599 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/json_marshal - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "2599 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack",
            "value": 773716,
            "unit": "ns/op\t 175.29 MB/s\t  524379 B/op\t      15 allocs/op",
            "extra": "3040 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack - ns/op",
            "value": 773716,
            "unit": "ns/op",
            "extra": "3040 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack - MB/s",
            "value": 175.29,
            "unit": "MB/s",
            "extra": "3040 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack - B/op",
            "value": 524379,
            "unit": "B/op",
            "extra": "3040 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/msgpack - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "3040 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast",
            "value": 254893,
            "unit": "ns/op\t 504.65 MB/s\t  131195 B/op\t       3 allocs/op",
            "extra": "9296 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast - ns/op",
            "value": 254893,
            "unit": "ns/op",
            "extra": "9296 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast - MB/s",
            "value": 504.65,
            "unit": "MB/s",
            "extra": "9296 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast - B/op",
            "value": 131195,
            "unit": "B/op",
            "extra": "9296 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "9296 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense",
            "value": 158344,
            "unit": "ns/op\t 237.27 MB/s\t   42356 B/op\t      10 allocs/op",
            "extra": "15166 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense - ns/op",
            "value": 158344,
            "unit": "ns/op",
            "extra": "15166 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense - MB/s",
            "value": 237.27,
            "unit": "MB/s",
            "extra": "15166 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense - B/op",
            "value": 42356,
            "unit": "B/op",
            "extra": "15166 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Wide_x1000/Wide1k/qdf_dense - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "15166 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json",
            "value": 916271,
            "unit": "ns/op\t 269.46 MB/s\t   48348 B/op\t    1001 allocs/op",
            "extra": "2617 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json - ns/op",
            "value": 916271,
            "unit": "ns/op",
            "extra": "2617 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json - MB/s",
            "value": 269.46,
            "unit": "MB/s",
            "extra": "2617 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json - B/op",
            "value": 48348,
            "unit": "B/op",
            "extra": "2617 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json - allocs/op",
            "value": 1001,
            "unit": "allocs/op",
            "extra": "2617 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal",
            "value": 959495,
            "unit": "ns/op\t 257.32 MB/s\t  302495 B/op\t    1002 allocs/op",
            "extra": "2506 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal - ns/op",
            "value": 959495,
            "unit": "ns/op",
            "extra": "2506 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal - MB/s",
            "value": 257.32,
            "unit": "MB/s",
            "extra": "2506 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal - B/op",
            "value": 302495,
            "unit": "B/op",
            "extra": "2506 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/json_marshal - allocs/op",
            "value": 1002,
            "unit": "allocs/op",
            "extra": "2506 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack",
            "value": 573859,
            "unit": "ns/op\t 323.49 MB/s\t  548385 B/op\t    1015 allocs/op",
            "extra": "4158 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack - ns/op",
            "value": 573859,
            "unit": "ns/op",
            "extra": "4158 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack - MB/s",
            "value": 323.49,
            "unit": "MB/s",
            "extra": "4158 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack - B/op",
            "value": 548385,
            "unit": "B/op",
            "extra": "4158 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/msgpack - allocs/op",
            "value": 1015,
            "unit": "allocs/op",
            "extra": "4158 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast",
            "value": 183005,
            "unit": "ns/op\t1014.45 MB/s\t  189045 B/op\t       3 allocs/op",
            "extra": "13075 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast - ns/op",
            "value": 183005,
            "unit": "ns/op",
            "extra": "13075 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast - MB/s",
            "value": 1014.45,
            "unit": "MB/s",
            "extra": "13075 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast - B/op",
            "value": 189045,
            "unit": "B/op",
            "extra": "13075 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "13075 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense",
            "value": 181441,
            "unit": "ns/op\t1023.19 MB/s\t  188903 B/op\t       3 allocs/op",
            "extra": "13215 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense - ns/op",
            "value": 181441,
            "unit": "ns/op",
            "extra": "13215 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense - MB/s",
            "value": 1023.19,
            "unit": "MB/s",
            "extra": "13215 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense - B/op",
            "value": 188903,
            "unit": "B/op",
            "extra": "13215 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k/LogBatch1k/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "13215 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json",
            "value": 758.4,
            "unit": "ns/op\t  31.64 MB/s\t     248 B/op\t       6 allocs/op",
            "extra": "3145210 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json - ns/op",
            "value": 758.4,
            "unit": "ns/op",
            "extra": "3145210 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json - MB/s",
            "value": 31.64,
            "unit": "MB/s",
            "extra": "3145210 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json - B/op",
            "value": 248,
            "unit": "B/op",
            "extra": "3145210 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/json - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "3145210 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack",
            "value": 333.9,
            "unit": "ns/op\t  47.92 MB/s\t      77 B/op\t       3 allocs/op",
            "extra": "7136770 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack - ns/op",
            "value": 333.9,
            "unit": "ns/op",
            "extra": "7136770 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack - MB/s",
            "value": 47.92,
            "unit": "MB/s",
            "extra": "7136770 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack - B/op",
            "value": 77,
            "unit": "B/op",
            "extra": "7136770 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/msgpack - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "7136770 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast",
            "value": 157.8,
            "unit": "ns/op\t 139.44 MB/s\t      29 B/op\t       2 allocs/op",
            "extra": "15238028 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast - ns/op",
            "value": 157.8,
            "unit": "ns/op",
            "extra": "15238028 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast - MB/s",
            "value": 139.44,
            "unit": "MB/s",
            "extra": "15238028 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast - B/op",
            "value": 29,
            "unit": "B/op",
            "extra": "15238028 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_fast - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "15238028 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense",
            "value": 360,
            "unit": "ns/op\t  69.44 MB/s\t      72 B/op\t       4 allocs/op",
            "extra": "6659576 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense - ns/op",
            "value": 360,
            "unit": "ns/op",
            "extra": "6659576 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense - MB/s",
            "value": 69.44,
            "unit": "MB/s",
            "extra": "6659576 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense - B/op",
            "value": 72,
            "unit": "B/op",
            "extra": "6659576 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Tiny/Tiny/qdf_dense - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "6659576 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json",
            "value": 4609,
            "unit": "ns/op\t  45.56 MB/s\t     448 B/op\t      10 allocs/op",
            "extra": "505210 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json - ns/op",
            "value": 4609,
            "unit": "ns/op",
            "extra": "505210 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json - MB/s",
            "value": 45.56,
            "unit": "MB/s",
            "extra": "505210 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json - B/op",
            "value": 448,
            "unit": "B/op",
            "extra": "505210 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/json - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "505210 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack",
            "value": 1595,
            "unit": "ns/op\t  84.01 MB/s\t     272 B/op\t       7 allocs/op",
            "extra": "1500273 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack - ns/op",
            "value": 1595,
            "unit": "ns/op",
            "extra": "1500273 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack - MB/s",
            "value": 84.01,
            "unit": "MB/s",
            "extra": "1500273 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack - B/op",
            "value": 272,
            "unit": "B/op",
            "extra": "1500273 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/msgpack - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "1500273 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast",
            "value": 816.2,
            "unit": "ns/op\t 161.72 MB/s\t     224 B/op\t       6 allocs/op",
            "extra": "2946583 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast - ns/op",
            "value": 816.2,
            "unit": "ns/op",
            "extra": "2946583 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast - MB/s",
            "value": 161.72,
            "unit": "MB/s",
            "extra": "2946583 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast - B/op",
            "value": 224,
            "unit": "B/op",
            "extra": "2946583 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_fast - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "2946583 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense",
            "value": 1453,
            "unit": "ns/op\t  94.98 MB/s\t     624 B/op\t       8 allocs/op",
            "extra": "1656246 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense - ns/op",
            "value": 1453,
            "unit": "ns/op",
            "extra": "1656246 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense - MB/s",
            "value": 94.98,
            "unit": "MB/s",
            "extra": "1656246 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense - B/op",
            "value": 624,
            "unit": "B/op",
            "extra": "1656246 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Flat/Flat/qdf_dense - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "1656246 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json",
            "value": 2558,
            "unit": "ns/op\t  40.27 MB/s\t     664 B/op\t      15 allocs/op",
            "extra": "889759 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json - ns/op",
            "value": 2558,
            "unit": "ns/op",
            "extra": "889759 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json - MB/s",
            "value": 40.27,
            "unit": "MB/s",
            "extra": "889759 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json - B/op",
            "value": 664,
            "unit": "B/op",
            "extra": "889759 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/json - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "889759 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack",
            "value": 1030,
            "unit": "ns/op\t  73.79 MB/s\t     160 B/op\t       6 allocs/op",
            "extra": "2318054 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack - ns/op",
            "value": 1030,
            "unit": "ns/op",
            "extra": "2318054 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack - MB/s",
            "value": 73.79,
            "unit": "MB/s",
            "extra": "2318054 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack - B/op",
            "value": 160,
            "unit": "B/op",
            "extra": "2318054 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/msgpack - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "2318054 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast",
            "value": 397.6,
            "unit": "ns/op\t 216.31 MB/s\t     112 B/op\t       5 allocs/op",
            "extra": "6063435 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast - ns/op",
            "value": 397.6,
            "unit": "ns/op",
            "extra": "6063435 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast - MB/s",
            "value": 216.31,
            "unit": "MB/s",
            "extra": "6063435 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast - B/op",
            "value": 112,
            "unit": "B/op",
            "extra": "6063435 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_fast - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "6063435 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense",
            "value": 954.2,
            "unit": "ns/op\t 100.60 MB/s\t     296 B/op\t      15 allocs/op",
            "extra": "2521983 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense - ns/op",
            "value": 954.2,
            "unit": "ns/op",
            "extra": "2521983 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense - MB/s",
            "value": 100.6,
            "unit": "MB/s",
            "extra": "2521983 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense - B/op",
            "value": 296,
            "unit": "B/op",
            "extra": "2521983 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Nested/Nested/qdf_dense - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "2521983 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json",
            "value": 7019,
            "unit": "ns/op\t  34.05 MB/s\t    1200 B/op\t      29 allocs/op",
            "extra": "332788 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json - ns/op",
            "value": 7019,
            "unit": "ns/op",
            "extra": "332788 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json - MB/s",
            "value": 34.05,
            "unit": "MB/s",
            "extra": "332788 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json - B/op",
            "value": 1200,
            "unit": "B/op",
            "extra": "332788 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/json - allocs/op",
            "value": 29,
            "unit": "allocs/op",
            "extra": "332788 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack",
            "value": 3746,
            "unit": "ns/op\t  37.10 MB/s\t     312 B/op\t      18 allocs/op",
            "extra": "650468 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack - ns/op",
            "value": 3746,
            "unit": "ns/op",
            "extra": "650468 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack - MB/s",
            "value": 37.1,
            "unit": "MB/s",
            "extra": "650468 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack - B/op",
            "value": 312,
            "unit": "B/op",
            "extra": "650468 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "650468 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast",
            "value": 1989,
            "unit": "ns/op\t  83.44 MB/s\t     264 B/op\t      17 allocs/op",
            "extra": "1206862 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast - ns/op",
            "value": 1989,
            "unit": "ns/op",
            "extra": "1206862 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast - MB/s",
            "value": 83.44,
            "unit": "MB/s",
            "extra": "1206862 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast - B/op",
            "value": 264,
            "unit": "B/op",
            "extra": "1206862 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_fast - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "1206862 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense",
            "value": 1958,
            "unit": "ns/op\t  32.17 MB/s\t     304 B/op\t      19 allocs/op",
            "extra": "1225128 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense - ns/op",
            "value": 1958,
            "unit": "ns/op",
            "extra": "1225128 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense - MB/s",
            "value": 32.17,
            "unit": "MB/s",
            "extra": "1225128 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense - B/op",
            "value": 304,
            "unit": "B/op",
            "extra": "1225128 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Deep16/Deep16/qdf_dense - allocs/op",
            "value": 19,
            "unit": "allocs/op",
            "extra": "1225128 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/json",
            "value": 4569550,
            "unit": "ns/op\t  46.59 MB/s\t  638351 B/op\t    5020 allocs/op",
            "extra": "511 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/json - ns/op",
            "value": 4569550,
            "unit": "ns/op",
            "extra": "511 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/json - MB/s",
            "value": 46.59,
            "unit": "MB/s",
            "extra": "511 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/json - B/op",
            "value": 638351,
            "unit": "B/op",
            "extra": "511 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/json - allocs/op",
            "value": 5020,
            "unit": "allocs/op",
            "extra": "511 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack",
            "value": 1535952,
            "unit": "ns/op\t  88.30 MB/s\t  409044 B/op\t    5007 allocs/op",
            "extra": "1545 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack - ns/op",
            "value": 1535952,
            "unit": "ns/op",
            "extra": "1545 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack - MB/s",
            "value": 88.3,
            "unit": "MB/s",
            "extra": "1545 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack - B/op",
            "value": 409044,
            "unit": "B/op",
            "extra": "1545 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/msgpack - allocs/op",
            "value": 5007,
            "unit": "allocs/op",
            "extra": "1545 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast",
            "value": 736209,
            "unit": "ns/op\t 174.72 MB/s\t  220500 B/op\t    5003 allocs/op",
            "extra": "3285 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast - ns/op",
            "value": 736209,
            "unit": "ns/op",
            "extra": "3285 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast - MB/s",
            "value": 174.72,
            "unit": "MB/s",
            "extra": "3285 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast - B/op",
            "value": 220500,
            "unit": "B/op",
            "extra": "3285 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_fast - allocs/op",
            "value": 5003,
            "unit": "allocs/op",
            "extra": "3285 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_dense",
            "value": 205246,
            "unit": "ns/op\t 183.05 MB/s\t  318265 B/op\t    5022 allocs/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_dense - ns/op",
            "value": 205246,
            "unit": "ns/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Wide_x1000/Wide1k/qdf_dense - MB/s",
            "value": 183.05,
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
            "value": 3466139,
            "unit": "ns/op\t  71.23 MB/s\t  442536 B/op\t    7019 allocs/op",
            "extra": "698 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/json - ns/op",
            "value": 3466139,
            "unit": "ns/op",
            "extra": "698 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/json - MB/s",
            "value": 71.23,
            "unit": "MB/s",
            "extra": "698 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/json - B/op",
            "value": 442536,
            "unit": "B/op",
            "extra": "698 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/json - allocs/op",
            "value": 7019,
            "unit": "allocs/op",
            "extra": "698 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack",
            "value": 1098010,
            "unit": "ns/op\t 169.07 MB/s\t  407512 B/op\t    7007 allocs/op",
            "extra": "2180 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack - ns/op",
            "value": 1098010,
            "unit": "ns/op",
            "extra": "2180 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack - MB/s",
            "value": 169.07,
            "unit": "MB/s",
            "extra": "2180 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack - B/op",
            "value": 407512,
            "unit": "B/op",
            "extra": "2180 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/msgpack - allocs/op",
            "value": 7007,
            "unit": "allocs/op",
            "extra": "2180 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast",
            "value": 424694,
            "unit": "ns/op\t 437.14 MB/s\t  251713 B/op\t    7002 allocs/op",
            "extra": "5594 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast - ns/op",
            "value": 424694,
            "unit": "ns/op",
            "extra": "5594 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast - MB/s",
            "value": 437.14,
            "unit": "MB/s",
            "extra": "5594 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast - B/op",
            "value": 251713,
            "unit": "B/op",
            "extra": "5594 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_fast - allocs/op",
            "value": 7002,
            "unit": "allocs/op",
            "extra": "5594 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense",
            "value": 425437,
            "unit": "ns/op\t 436.37 MB/s\t  255170 B/op\t    7005 allocs/op",
            "extra": "5707 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense - ns/op",
            "value": 425437,
            "unit": "ns/op",
            "extra": "5707 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense - MB/s",
            "value": 436.37,
            "unit": "MB/s",
            "extra": "5707 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense - B/op",
            "value": 255170,
            "unit": "B/op",
            "extra": "5707 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k/LogBatch1k/qdf_dense - allocs/op",
            "value": 7005,
            "unit": "allocs/op",
            "extra": "5707 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen",
            "value": 323406,
            "unit": "ns/op\t 574.04 MB/s\t  908027 B/op\t      26 allocs/op",
            "extra": "7544 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen - ns/op",
            "value": 323406,
            "unit": "ns/op",
            "extra": "7544 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen - MB/s",
            "value": 574.04,
            "unit": "MB/s",
            "extra": "7544 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen - B/op",
            "value": 908027,
            "unit": "B/op",
            "extra": "7544 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_LogBatch1k_Codegen - allocs/op",
            "value": 26,
            "unit": "allocs/op",
            "extra": "7544 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen",
            "value": 422729,
            "unit": "ns/op\t 439.17 MB/s\t  251648 B/op\t    7001 allocs/op",
            "extra": "5432 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen - ns/op",
            "value": 422729,
            "unit": "ns/op",
            "extra": "5432 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen - MB/s",
            "value": 439.17,
            "unit": "MB/s",
            "extra": "5432 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen - B/op",
            "value": 251648,
            "unit": "B/op",
            "extra": "5432 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_LogBatch1k_Codegen - allocs/op",
            "value": 7001,
            "unit": "allocs/op",
            "extra": "5432 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json",
            "value": 173870,
            "unit": "ns/op\t 154.86 MB/s\t   27432 B/op\t       2 allocs/op",
            "extra": "13773 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json - ns/op",
            "value": 173870,
            "unit": "ns/op",
            "extra": "13773 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json - MB/s",
            "value": 154.86,
            "unit": "MB/s",
            "extra": "13773 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json - B/op",
            "value": 27432,
            "unit": "B/op",
            "extra": "13773 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "13773 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack",
            "value": 179930,
            "unit": "ns/op\t 211.16 MB/s\t  131235 B/op\t      13 allocs/op",
            "extra": "13315 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack - ns/op",
            "value": 179930,
            "unit": "ns/op",
            "extra": "13315 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack - MB/s",
            "value": 211.16,
            "unit": "MB/s",
            "extra": "13315 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack - B/op",
            "value": 131235,
            "unit": "B/op",
            "extra": "13315 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/msgpack - allocs/op",
            "value": 13,
            "unit": "allocs/op",
            "extra": "13315 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf",
            "value": 15024,
            "unit": "ns/op\t 583.01 MB/s\t    9794 B/op\t       3 allocs/op",
            "extra": "158425 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf - ns/op",
            "value": 15024,
            "unit": "ns/op",
            "extra": "158425 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf - MB/s",
            "value": 583.01,
            "unit": "MB/s",
            "extra": "158425 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf - B/op",
            "value": 9794,
            "unit": "B/op",
            "extra": "158425 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "158425 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json",
            "value": 628341,
            "unit": "ns/op\t  42.85 MB/s\t  104576 B/op\t      65 allocs/op",
            "extra": "3733 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json - ns/op",
            "value": 628341,
            "unit": "ns/op",
            "extra": "3733 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json - MB/s",
            "value": 42.85,
            "unit": "MB/s",
            "extra": "3733 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json - B/op",
            "value": 104576,
            "unit": "B/op",
            "extra": "3733 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/json - allocs/op",
            "value": 65,
            "unit": "allocs/op",
            "extra": "3733 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack",
            "value": 226736,
            "unit": "ns/op\t 167.57 MB/s\t   68193 B/op\t      29 allocs/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack - ns/op",
            "value": 226736,
            "unit": "ns/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack - MB/s",
            "value": 167.57,
            "unit": "MB/s",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack - B/op",
            "value": 68193,
            "unit": "B/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/msgpack - allocs/op",
            "value": 29,
            "unit": "allocs/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf",
            "value": 12586,
            "unit": "ns/op\t 695.92 MB/s\t   42333 B/op\t      11 allocs/op",
            "extra": "185860 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf - ns/op",
            "value": 12586,
            "unit": "ns/op",
            "extra": "185860 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf - MB/s",
            "value": 695.92,
            "unit": "MB/s",
            "extra": "185860 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf - B/op",
            "value": 42333,
            "unit": "B/op",
            "extra": "185860 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_WebRequest/webreq_1024/decode/qdf - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "185860 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json",
            "value": 88492,
            "unit": "ns/op\t 195.66 MB/s\t   18533 B/op\t       2 allocs/op",
            "extra": "27034 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json - ns/op",
            "value": 88492,
            "unit": "ns/op",
            "extra": "27034 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json - MB/s",
            "value": 195.66,
            "unit": "MB/s",
            "extra": "27034 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json - B/op",
            "value": 18533,
            "unit": "B/op",
            "extra": "27034 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "27034 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack",
            "value": 109640,
            "unit": "ns/op\t 252.74 MB/s\t   65625 B/op\t      12 allocs/op",
            "extra": "21926 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack - ns/op",
            "value": 109640,
            "unit": "ns/op",
            "extra": "21926 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack - MB/s",
            "value": 252.74,
            "unit": "MB/s",
            "extra": "21926 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack - B/op",
            "value": 65625,
            "unit": "B/op",
            "extra": "21926 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/msgpack - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "21926 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf",
            "value": 12303,
            "unit": "ns/op\t  45.76 MB/s\t     768 B/op\t       3 allocs/op",
            "extra": "192806 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf - ns/op",
            "value": 12303,
            "unit": "ns/op",
            "extra": "192806 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf - MB/s",
            "value": 45.76,
            "unit": "MB/s",
            "extra": "192806 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf - B/op",
            "value": 768,
            "unit": "B/op",
            "extra": "192806 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "192806 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json",
            "value": 387568,
            "unit": "ns/op\t  44.67 MB/s\t   75976 B/op\t      43 allocs/op",
            "extra": "6060 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json - ns/op",
            "value": 387568,
            "unit": "ns/op",
            "extra": "6060 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json - MB/s",
            "value": 44.67,
            "unit": "MB/s",
            "extra": "6060 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json - B/op",
            "value": 75976,
            "unit": "B/op",
            "extra": "6060 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/json - allocs/op",
            "value": 43,
            "unit": "allocs/op",
            "extra": "6060 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack",
            "value": 146894,
            "unit": "ns/op\t 188.64 MB/s\t   49543 B/op\t      18 allocs/op",
            "extra": "16350 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack - ns/op",
            "value": 146894,
            "unit": "ns/op",
            "extra": "16350 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack - MB/s",
            "value": 188.64,
            "unit": "MB/s",
            "extra": "16350 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack - B/op",
            "value": 49543,
            "unit": "B/op",
            "extra": "16350 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "16350 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf",
            "value": 10153,
            "unit": "ns/op\t  55.45 MB/s\t   32895 B/op\t       6 allocs/op",
            "extra": "220424 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf - ns/op",
            "value": 10153,
            "unit": "ns/op",
            "extra": "220424 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf - MB/s",
            "value": 55.45,
            "unit": "MB/s",
            "extra": "220424 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf - B/op",
            "value": 32895,
            "unit": "B/op",
            "extra": "220424 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Counters/counters_1024/decode/qdf - allocs/op",
            "value": 6,
            "unit": "allocs/op",
            "extra": "220424 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json",
            "value": 34141,
            "unit": "ns/op\t 199.97 MB/s\t    6961 B/op\t       2 allocs/op",
            "extra": "69787 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json - ns/op",
            "value": 34141,
            "unit": "ns/op",
            "extra": "69787 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json - MB/s",
            "value": 199.97,
            "unit": "MB/s",
            "extra": "69787 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json - B/op",
            "value": 6961,
            "unit": "B/op",
            "extra": "69787 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "69787 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack",
            "value": 39201,
            "unit": "ns/op\t 235.94 MB/s\t   32804 B/op\t      11 allocs/op",
            "extra": "60654 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack - ns/op",
            "value": 39201,
            "unit": "ns/op",
            "extra": "60654 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack - MB/s",
            "value": 235.94,
            "unit": "MB/s",
            "extra": "60654 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack - B/op",
            "value": 32804,
            "unit": "B/op",
            "extra": "60654 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/msgpack - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "60654 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf",
            "value": 11839,
            "unit": "ns/op\t  26.02 MB/s\t     416 B/op\t       3 allocs/op",
            "extra": "202448 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf - ns/op",
            "value": 11839,
            "unit": "ns/op",
            "extra": "202448 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf - MB/s",
            "value": 26.02,
            "unit": "MB/s",
            "extra": "202448 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf - B/op",
            "value": 416,
            "unit": "B/op",
            "extra": "202448 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "202448 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json",
            "value": 148944,
            "unit": "ns/op\t  45.84 MB/s\t   25504 B/op\t      19 allocs/op",
            "extra": "16100 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json - ns/op",
            "value": 148944,
            "unit": "ns/op",
            "extra": "16100 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json - MB/s",
            "value": 45.84,
            "unit": "MB/s",
            "extra": "16100 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json - B/op",
            "value": 25504,
            "unit": "B/op",
            "extra": "16100 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/json - allocs/op",
            "value": 19,
            "unit": "allocs/op",
            "extra": "16100 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack",
            "value": 49215,
            "unit": "ns/op\t 187.93 MB/s\t   16570 B/op\t       8 allocs/op",
            "extra": "48824 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack - ns/op",
            "value": 49215,
            "unit": "ns/op",
            "extra": "48824 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack - MB/s",
            "value": 187.93,
            "unit": "MB/s",
            "extra": "48824 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack - B/op",
            "value": 16570,
            "unit": "B/op",
            "extra": "48824 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/msgpack - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "48824 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf",
            "value": 5795,
            "unit": "ns/op\t  53.15 MB/s\t   16452 B/op\t       4 allocs/op",
            "extra": "406440 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf - ns/op",
            "value": 5795,
            "unit": "ns/op",
            "extra": "406440 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf - MB/s",
            "value": 53.15,
            "unit": "MB/s",
            "extra": "406440 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf - B/op",
            "value": 16452,
            "unit": "B/op",
            "extra": "406440 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_SpreadEnum/spread_enum_1024/decode/qdf - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "406440 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json",
            "value": 173225,
            "unit": "ns/op\t 424.04 MB/s\t   73801 B/op\t       2 allocs/op",
            "extra": "13856 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json - ns/op",
            "value": 173225,
            "unit": "ns/op",
            "extra": "13856 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json - MB/s",
            "value": 424.04,
            "unit": "MB/s",
            "extra": "13856 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json - B/op",
            "value": 73801,
            "unit": "B/op",
            "extra": "13856 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "13856 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack",
            "value": 146629,
            "unit": "ns/op\t 407.23 MB/s\t  131100 B/op\t      13 allocs/op",
            "extra": "16372 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack - ns/op",
            "value": 146629,
            "unit": "ns/op",
            "extra": "16372 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack - MB/s",
            "value": 407.23,
            "unit": "MB/s",
            "extra": "16372 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack - B/op",
            "value": 131100,
            "unit": "B/op",
            "extra": "16372 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/msgpack - allocs/op",
            "value": 13,
            "unit": "allocs/op",
            "extra": "16372 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf",
            "value": 87597,
            "unit": "ns/op\t 383.65 MB/s\t   41121 B/op\t       3 allocs/op",
            "extra": "27441 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf - ns/op",
            "value": 87597,
            "unit": "ns/op",
            "extra": "27441 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf - MB/s",
            "value": 383.65,
            "unit": "MB/s",
            "extra": "27441 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf - B/op",
            "value": 41121,
            "unit": "B/op",
            "extra": "27441 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "27441 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json",
            "value": 982565,
            "unit": "ns/op\t  74.76 MB/s\t  125256 B/op\t    2016 allocs/op",
            "extra": "2253 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json - ns/op",
            "value": 982565,
            "unit": "ns/op",
            "extra": "2253 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json - MB/s",
            "value": 74.76,
            "unit": "MB/s",
            "extra": "2253 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json - B/op",
            "value": 125256,
            "unit": "B/op",
            "extra": "2253 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/json - allocs/op",
            "value": 2016,
            "unit": "allocs/op",
            "extra": "2253 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack",
            "value": 301981,
            "unit": "ns/op\t 197.73 MB/s\t  114785 B/op\t    2007 allocs/op",
            "extra": "7521 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack - ns/op",
            "value": 301981,
            "unit": "ns/op",
            "extra": "7521 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack - MB/s",
            "value": 197.73,
            "unit": "MB/s",
            "extra": "7521 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack - B/op",
            "value": 114785,
            "unit": "B/op",
            "extra": "7521 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/msgpack - allocs/op",
            "value": 2007,
            "unit": "allocs/op",
            "extra": "7521 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf",
            "value": 100700,
            "unit": "ns/op\t 333.73 MB/s\t   65210 B/op\t    1012 allocs/op",
            "extra": "23872 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf - ns/op",
            "value": 100700,
            "unit": "ns/op",
            "extra": "23872 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf - MB/s",
            "value": 333.73,
            "unit": "MB/s",
            "extra": "23872 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf - B/op",
            "value": 65210,
            "unit": "B/op",
            "extra": "23872 times\n4 procs"
          },
          {
            "name": "BenchmarkCorpusCodec_Traces/traces_500/decode/qdf - allocs/op",
            "value": 1012,
            "unit": "allocs/op",
            "extra": "23872 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/json",
            "value": 28632,
            "unit": "ns/op\t    2736 B/op\t       2 allocs/op",
            "extra": "83631 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/json - ns/op",
            "value": 28632,
            "unit": "ns/op",
            "extra": "83631 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/json - B/op",
            "value": 2736,
            "unit": "B/op",
            "extra": "83631 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "83631 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/msgpack",
            "value": 17245,
            "unit": "ns/op\t    8225 B/op\t       9 allocs/op",
            "extra": "138391 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/msgpack - ns/op",
            "value": 17245,
            "unit": "ns/op",
            "extra": "138391 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/msgpack - B/op",
            "value": 8225,
            "unit": "B/op",
            "extra": "138391 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/msgpack - allocs/op",
            "value": 9,
            "unit": "allocs/op",
            "extra": "138391 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/qdf_fast",
            "value": 1904,
            "unit": "ns/op\t    2784 B/op\t       3 allocs/op",
            "extra": "1257512 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/qdf_fast - ns/op",
            "value": 1904,
            "unit": "ns/op",
            "extra": "1257512 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/qdf_fast - B/op",
            "value": 2784,
            "unit": "B/op",
            "extra": "1257512 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float32Vec512/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1257512 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/json",
            "value": 30315,
            "unit": "ns/op\t    2736 B/op\t       2 allocs/op",
            "extra": "78891 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/json - ns/op",
            "value": 30315,
            "unit": "ns/op",
            "extra": "78891 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/json - B/op",
            "value": 2736,
            "unit": "B/op",
            "extra": "78891 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "78891 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/msgpack",
            "value": 20664,
            "unit": "ns/op\t   16418 B/op\t      10 allocs/op",
            "extra": "115957 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/msgpack - ns/op",
            "value": 20664,
            "unit": "ns/op",
            "extra": "115957 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/msgpack - B/op",
            "value": 16418,
            "unit": "B/op",
            "extra": "115957 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/msgpack - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "115957 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/qdf_fast",
            "value": 2246,
            "unit": "ns/op\t    4961 B/op\t       3 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_Float64Vec512/qdf_fast - ns/op",
            "value": 2246,
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
            "value": 74875,
            "unit": "ns/op\t    4384 B/op\t      16 allocs/op",
            "extra": "31963 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/json - ns/op",
            "value": 74875,
            "unit": "ns/op",
            "extra": "31963 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/json - B/op",
            "value": 4384,
            "unit": "B/op",
            "extra": "31963 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/json - allocs/op",
            "value": 16,
            "unit": "allocs/op",
            "extra": "31963 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/msgpack",
            "value": 23837,
            "unit": "ns/op\t    4280 B/op\t       8 allocs/op",
            "extra": "100588 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/msgpack - ns/op",
            "value": 23837,
            "unit": "ns/op",
            "extra": "100588 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/msgpack - B/op",
            "value": 4280,
            "unit": "B/op",
            "extra": "100588 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/msgpack - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "100588 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/qdf_fast",
            "value": 3964,
            "unit": "ns/op\t    2112 B/op\t       3 allocs/op",
            "extra": "609505 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/qdf_fast - ns/op",
            "value": 3964,
            "unit": "ns/op",
            "extra": "609505 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/qdf_fast - B/op",
            "value": 2112,
            "unit": "B/op",
            "extra": "609505 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_Float32Vec512/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "609505 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/json",
            "value": 159193338,
            "unit": "ns/op\t 233.91 MB/s\t58506766 B/op\t  350217 allocs/op",
            "extra": "13 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/json - ns/op",
            "value": 159193338,
            "unit": "ns/op",
            "extra": "13 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/json - MB/s",
            "value": 233.91,
            "unit": "MB/s",
            "extra": "13 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/json - B/op",
            "value": 58506766,
            "unit": "B/op",
            "extra": "13 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/json - allocs/op",
            "value": 350217,
            "unit": "allocs/op",
            "extra": "13 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/msgpack",
            "value": 85676261,
            "unit": "ns/op\t 284.43 MB/s\t68709109 B/op\t  100022 allocs/op",
            "extra": "27 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/msgpack - ns/op",
            "value": 85676261,
            "unit": "ns/op",
            "extra": "27 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/msgpack - MB/s",
            "value": 284.43,
            "unit": "MB/s",
            "extra": "27 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/msgpack - B/op",
            "value": 68709109,
            "unit": "B/op",
            "extra": "27 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/msgpack - allocs/op",
            "value": 100022,
            "unit": "allocs/op",
            "extra": "27 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_fast",
            "value": 27491132,
            "unit": "ns/op\t 876.92 MB/s\t29512037 B/op\t      19 allocs/op",
            "extra": "80 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_fast - ns/op",
            "value": 27491132,
            "unit": "ns/op",
            "extra": "80 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_fast - MB/s",
            "value": 876.92,
            "unit": "MB/s",
            "extra": "80 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_fast - B/op",
            "value": 29512037,
            "unit": "B/op",
            "extra": "80 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_fast - allocs/op",
            "value": 19,
            "unit": "allocs/op",
            "extra": "80 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_qpack",
            "value": 24507083,
            "unit": "ns/op\t 958.71 MB/s\t28815726 B/op\t      19 allocs/op",
            "extra": "100 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_qpack - ns/op",
            "value": 24507083,
            "unit": "ns/op",
            "extra": "100 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_qpack - MB/s",
            "value": 958.71,
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
            "value": 26767175,
            "unit": "ns/op\t 675.67 MB/s\t24114199 B/op\t      74 allocs/op",
            "extra": "90 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_dense - ns/op",
            "value": 26767175,
            "unit": "ns/op",
            "extra": "90 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_dense - MB/s",
            "value": 675.67,
            "unit": "MB/s",
            "extra": "90 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_dense - B/op",
            "value": 24114199,
            "unit": "B/op",
            "extra": "90 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Encode/qdf_dense - allocs/op",
            "value": 74,
            "unit": "allocs/op",
            "extra": "90 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/json",
            "value": 619589272,
            "unit": "ns/op\t  60.11 MB/s\t119804004 B/op\t 1559637 allocs/op",
            "extra": "4 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/json - ns/op",
            "value": 619589272,
            "unit": "ns/op",
            "extra": "4 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/json - MB/s",
            "value": 60.11,
            "unit": "MB/s",
            "extra": "4 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/json - B/op",
            "value": 119804004,
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
            "value": 184938170,
            "unit": "ns/op\t 131.79 MB/s\t74391124 B/op\t 1425125 allocs/op",
            "extra": "13 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/msgpack - ns/op",
            "value": 184938170,
            "unit": "ns/op",
            "extra": "13 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/msgpack - MB/s",
            "value": 131.79,
            "unit": "MB/s",
            "extra": "13 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/msgpack - B/op",
            "value": 74391124,
            "unit": "B/op",
            "extra": "13 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/msgpack - allocs/op",
            "value": 1425125,
            "unit": "allocs/op",
            "extra": "13 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_fast",
            "value": 71355193,
            "unit": "ns/op\t 337.90 MB/s\t48379658 B/op\t  875099 allocs/op",
            "extra": "33 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_fast - ns/op",
            "value": 71355193,
            "unit": "ns/op",
            "extra": "33 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_fast - MB/s",
            "value": 337.9,
            "unit": "MB/s",
            "extra": "33 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_fast - B/op",
            "value": 48379658,
            "unit": "B/op",
            "extra": "33 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_fast - allocs/op",
            "value": 875099,
            "unit": "allocs/op",
            "extra": "33 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_qpack",
            "value": 67101290,
            "unit": "ns/op\t 350.18 MB/s\t48380685 B/op\t  875099 allocs/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_qpack - ns/op",
            "value": 67101290,
            "unit": "ns/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_qpack - MB/s",
            "value": 350.18,
            "unit": "MB/s",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_qpack - B/op",
            "value": 48380685,
            "unit": "B/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_qpack - allocs/op",
            "value": 875099,
            "unit": "allocs/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense",
            "value": 62406693,
            "unit": "ns/op\t 289.83 MB/s\t50892035 B/op\t  790950 allocs/op",
            "extra": "37 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense - ns/op",
            "value": 62406693,
            "unit": "ns/op",
            "extra": "37 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense - MB/s",
            "value": 289.83,
            "unit": "MB/s",
            "extra": "37 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense - B/op",
            "value": 50892035,
            "unit": "B/op",
            "extra": "37 times\n4 procs"
          },
          {
            "name": "BenchmarkLargePayload_Decode/qdf_dense - allocs/op",
            "value": 790950,
            "unit": "allocs/op",
            "extra": "37 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/json",
            "value": 8025,
            "unit": "ns/op\t    3408 B/op\t      84 allocs/op",
            "extra": "294085 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/json - ns/op",
            "value": 8025,
            "unit": "ns/op",
            "extra": "294085 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/json - B/op",
            "value": 3408,
            "unit": "B/op",
            "extra": "294085 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/json - allocs/op",
            "value": 84,
            "unit": "allocs/op",
            "extra": "294085 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/msgpack",
            "value": 4329,
            "unit": "ns/op\t    1536 B/op\t      46 allocs/op",
            "extra": "546277 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/msgpack - ns/op",
            "value": 4329,
            "unit": "ns/op",
            "extra": "546277 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/msgpack - B/op",
            "value": 1536,
            "unit": "B/op",
            "extra": "546277 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/msgpack - allocs/op",
            "value": 46,
            "unit": "allocs/op",
            "extra": "546277 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_fast",
            "value": 1486,
            "unit": "ns/op\t     416 B/op\t       3 allocs/op",
            "extra": "1622036 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_fast - ns/op",
            "value": 1486,
            "unit": "ns/op",
            "extra": "1622036 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_fast - B/op",
            "value": 416,
            "unit": "B/op",
            "extra": "1622036 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1622036 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_dense",
            "value": 1706,
            "unit": "ns/op\t     416 B/op\t       3 allocs/op",
            "extra": "1408314 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_dense - ns/op",
            "value": 1706,
            "unit": "ns/op",
            "extra": "1408314 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_dense - B/op",
            "value": 416,
            "unit": "B/op",
            "extra": "1408314 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MapHeavy/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1408314 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/json",
            "value": 17033,
            "unit": "ns/op\t    4912 B/op\t     124 allocs/op",
            "extra": "140073 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/json - ns/op",
            "value": 17033,
            "unit": "ns/op",
            "extra": "140073 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/json - B/op",
            "value": 4912,
            "unit": "B/op",
            "extra": "140073 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/json - allocs/op",
            "value": 124,
            "unit": "allocs/op",
            "extra": "140073 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/msgpack",
            "value": 7530,
            "unit": "ns/op\t    3088 B/op\t     112 allocs/op",
            "extra": "284012 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/msgpack - ns/op",
            "value": 7530,
            "unit": "ns/op",
            "extra": "284012 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/msgpack - B/op",
            "value": 3088,
            "unit": "B/op",
            "extra": "284012 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/msgpack - allocs/op",
            "value": 112,
            "unit": "allocs/op",
            "extra": "284012 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/qdf_fast",
            "value": 2881,
            "unit": "ns/op\t    2354 B/op\t      32 allocs/op",
            "extra": "849302 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/qdf_fast - ns/op",
            "value": 2881,
            "unit": "ns/op",
            "extra": "849302 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/qdf_fast - B/op",
            "value": 2354,
            "unit": "B/op",
            "extra": "849302 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy/qdf_fast - allocs/op",
            "value": 32,
            "unit": "allocs/op",
            "extra": "849302 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/json",
            "value": 9207,
            "unit": "ns/op\t    2820 B/op\t      71 allocs/op",
            "extra": "260102 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/json - ns/op",
            "value": 9207,
            "unit": "ns/op",
            "extra": "260102 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/json - B/op",
            "value": 2820,
            "unit": "B/op",
            "extra": "260102 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/json - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "260102 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/msgpack",
            "value": 2343,
            "unit": "ns/op\t    1487 B/op\t      46 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/msgpack - ns/op",
            "value": 2343,
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
            "value": 1661,
            "unit": "ns/op\t    1403 B/op\t      25 allocs/op",
            "extra": "1453166 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/qdf_fast - ns/op",
            "value": 1661,
            "unit": "ns/op",
            "extra": "1453166 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/qdf_fast - B/op",
            "value": 1403,
            "unit": "B/op",
            "extra": "1453166 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapHeavy_RepeatedKeys/qdf_fast - allocs/op",
            "value": 25,
            "unit": "allocs/op",
            "extra": "1453166 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/json",
            "value": 0.3469,
            "unit": "ns/op\t    442537 B/decode\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/json - ns/op",
            "value": 0.3469,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/json - B/decode",
            "value": 442537,
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
            "value": 0.1119,
            "unit": "ns/op\t    407516 B/decode\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/msgpack - ns/op",
            "value": 0.1119,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/msgpack - B/decode",
            "value": 407516,
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
            "value": 0.04415,
            "unit": "ns/op\t    251762 B/decode\t       0 B/op\t       0 allocs/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/qdf_fast - ns/op",
            "value": 0.04415,
            "unit": "ns/op",
            "extra": "1000000000 times\n4 procs"
          },
          {
            "name": "BenchmarkMemory_DecodeLogBatch1k_Bytes/qdf_fast - B/decode",
            "value": 251762,
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
            "value": 3537,
            "unit": "ns/op\t     790 B/op\t      37 allocs/op",
            "extra": "650084 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/json - ns/op",
            "value": 3537,
            "unit": "ns/op",
            "extra": "650084 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/json - B/op",
            "value": 790,
            "unit": "B/op",
            "extra": "650084 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/json - allocs/op",
            "value": 37,
            "unit": "allocs/op",
            "extra": "650084 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/qdf_fast",
            "value": 600.2,
            "unit": "ns/op\t     345 B/op\t       3 allocs/op",
            "extra": "4009438 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/qdf_fast - ns/op",
            "value": 600.2,
            "unit": "ns/op",
            "extra": "4009438 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/qdf_fast - B/op",
            "value": 345,
            "unit": "B/op",
            "extra": "4009438 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_MapStringAny_RepeatedKeys/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "4009438 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json",
            "value": 638.4,
            "unit": "ns/op\t 151.93 MB/s\t     240 B/op\t       3 allocs/op",
            "extra": "3746180 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json - ns/op",
            "value": 638.4,
            "unit": "ns/op",
            "extra": "3746180 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json - MB/s",
            "value": 151.93,
            "unit": "MB/s",
            "extra": "3746180 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json - B/op",
            "value": 240,
            "unit": "B/op",
            "extra": "3746180 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/json - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3746180 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack",
            "value": 391.8,
            "unit": "ns/op\t 160.81 MB/s\t     192 B/op\t       3 allocs/op",
            "extra": "6123063 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack - ns/op",
            "value": 391.8,
            "unit": "ns/op",
            "extra": "6123063 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack - MB/s",
            "value": 160.81,
            "unit": "MB/s",
            "extra": "6123063 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack - B/op",
            "value": 192,
            "unit": "B/op",
            "extra": "6123063 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/msgpack - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "6123063 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf",
            "value": 392.9,
            "unit": "ns/op\t 183.25 MB/s\t     240 B/op\t       3 allocs/op",
            "extra": "6153187 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf - ns/op",
            "value": 392.9,
            "unit": "ns/op",
            "extra": "6153187 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf - MB/s",
            "value": 183.25,
            "unit": "MB/s",
            "extra": "6153187 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf - B/op",
            "value": 240,
            "unit": "B/op",
            "extra": "6153187 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "6153187 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json",
            "value": 1696,
            "unit": "ns/op\t  57.18 MB/s\t     328 B/op\t       7 allocs/op",
            "extra": "1414930 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json - ns/op",
            "value": 1696,
            "unit": "ns/op",
            "extra": "1414930 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json - MB/s",
            "value": 57.18,
            "unit": "MB/s",
            "extra": "1414930 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json - B/op",
            "value": 328,
            "unit": "B/op",
            "extra": "1414930 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/json - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "1414930 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack",
            "value": 634.6,
            "unit": "ns/op\t  99.27 MB/s\t     160 B/op\t       4 allocs/op",
            "extra": "3776937 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack - ns/op",
            "value": 634.6,
            "unit": "ns/op",
            "extra": "3776937 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack - MB/s",
            "value": 99.27,
            "unit": "MB/s",
            "extra": "3776937 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack - B/op",
            "value": 160,
            "unit": "B/op",
            "extra": "3776937 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/msgpack - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "3776937 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf",
            "value": 274.4,
            "unit": "ns/op\t 262.37 MB/s\t     112 B/op\t       3 allocs/op",
            "extra": "8757856 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf - ns/op",
            "value": 274.4,
            "unit": "ns/op",
            "extra": "8757856 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf - MB/s",
            "value": 262.37,
            "unit": "MB/s",
            "extra": "8757856 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf - B/op",
            "value": 112,
            "unit": "B/op",
            "extra": "8757856 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_HotPath/hot_path/decode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "8757856 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json",
            "value": 374067,
            "unit": "ns/op\t 381.97 MB/s\t  147922 B/op\t       2 allocs/op",
            "extra": "6286 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json - ns/op",
            "value": 374067,
            "unit": "ns/op",
            "extra": "6286 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json - MB/s",
            "value": 381.97,
            "unit": "MB/s",
            "extra": "6286 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json - B/op",
            "value": 147922,
            "unit": "B/op",
            "extra": "6286 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "6286 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack",
            "value": 450703,
            "unit": "ns/op\t 247.70 MB/s\t  286202 B/op\t    1014 allocs/op",
            "extra": "5295 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack - ns/op",
            "value": 450703,
            "unit": "ns/op",
            "extra": "5295 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack - MB/s",
            "value": 247.7,
            "unit": "MB/s",
            "extra": "5295 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack - B/op",
            "value": 286202,
            "unit": "B/op",
            "extra": "5295 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/msgpack - allocs/op",
            "value": 1014,
            "unit": "allocs/op",
            "extra": "5295 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf",
            "value": 251521,
            "unit": "ns/op\t 161.27 MB/s\t   41234 B/op\t       3 allocs/op",
            "extra": "9483 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf - ns/op",
            "value": 251521,
            "unit": "ns/op",
            "extra": "9483 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf - MB/s",
            "value": 161.27,
            "unit": "MB/s",
            "extra": "9483 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf - B/op",
            "value": 41234,
            "unit": "B/op",
            "extra": "9483 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "9483 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json",
            "value": 2623868,
            "unit": "ns/op\t  54.45 MB/s\t  503578 B/op\t    9019 allocs/op",
            "extra": "898 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json - ns/op",
            "value": 2623868,
            "unit": "ns/op",
            "extra": "898 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json - MB/s",
            "value": 54.45,
            "unit": "MB/s",
            "extra": "898 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json - B/op",
            "value": 503578,
            "unit": "B/op",
            "extra": "898 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/json - allocs/op",
            "value": 9019,
            "unit": "allocs/op",
            "extra": "898 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack",
            "value": 926209,
            "unit": "ns/op\t 120.53 MB/s\t  323890 B/op\t    8007 allocs/op",
            "extra": "2550 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack - ns/op",
            "value": 926209,
            "unit": "ns/op",
            "extra": "2550 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack - MB/s",
            "value": 120.53,
            "unit": "MB/s",
            "extra": "2550 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack - B/op",
            "value": 323890,
            "unit": "B/op",
            "extra": "2550 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/msgpack - allocs/op",
            "value": 8007,
            "unit": "allocs/op",
            "extra": "2550 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf",
            "value": 330197,
            "unit": "ns/op\t 122.84 MB/s\t  169218 B/op\t    3468 allocs/op",
            "extra": "7213 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf - ns/op",
            "value": 330197,
            "unit": "ns/op",
            "extra": "7213 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf - MB/s",
            "value": 122.84,
            "unit": "MB/s",
            "extra": "7213 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf - B/op",
            "value": 169218,
            "unit": "B/op",
            "extra": "7213 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch/telemetry_1k/decode/qdf - allocs/op",
            "value": 3468,
            "unit": "allocs/op",
            "extra": "7213 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf",
            "value": 249127,
            "unit": "ns/op\t 162.82 MB/s\t      91 B/op\t       2 allocs/op",
            "extra": "9540 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf - ns/op",
            "value": 249127,
            "unit": "ns/op",
            "extra": "9540 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf - MB/s",
            "value": 162.82,
            "unit": "MB/s",
            "extra": "9540 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf - B/op",
            "value": 91,
            "unit": "B/op",
            "extra": "9540 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_TelemetryBatch_PreIntern/telemetry_1k_preintern/encode/qdf - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "9540 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json",
            "value": 161955,
            "unit": "ns/op\t 230.05 MB/s\t   41115 B/op\t       2 allocs/op",
            "extra": "14828 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json - ns/op",
            "value": 161955,
            "unit": "ns/op",
            "extra": "14828 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json - MB/s",
            "value": 230.05,
            "unit": "MB/s",
            "extra": "14828 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json - B/op",
            "value": 41115,
            "unit": "B/op",
            "extra": "14828 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "14828 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack",
            "value": 104888,
            "unit": "ns/op\t 186.03 MB/s\t   65626 B/op\t      12 allocs/op",
            "extra": "22848 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack - ns/op",
            "value": 104888,
            "unit": "ns/op",
            "extra": "22848 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack - MB/s",
            "value": 186.03,
            "unit": "MB/s",
            "extra": "22848 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack - B/op",
            "value": 65626,
            "unit": "B/op",
            "extra": "22848 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/msgpack - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "22848 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf",
            "value": 4268,
            "unit": "ns/op\t1966.11 MB/s\t    9668 B/op\t       3 allocs/op",
            "extra": "561828 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf - ns/op",
            "value": 4268,
            "unit": "ns/op",
            "extra": "561828 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf - MB/s",
            "value": 1966.11,
            "unit": "MB/s",
            "extra": "561828 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf - B/op",
            "value": 9668,
            "unit": "B/op",
            "extra": "561828 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "561828 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json",
            "value": 550133,
            "unit": "ns/op\t  67.73 MB/s\t   54080 B/op\t      40 allocs/op",
            "extra": "4270 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json - ns/op",
            "value": 550133,
            "unit": "ns/op",
            "extra": "4270 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json - MB/s",
            "value": 67.73,
            "unit": "MB/s",
            "extra": "4270 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json - B/op",
            "value": 54080,
            "unit": "B/op",
            "extra": "4270 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/json - allocs/op",
            "value": 40,
            "unit": "allocs/op",
            "extra": "4270 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack",
            "value": 130486,
            "unit": "ns/op\t 149.53 MB/s\t   35197 B/op\t      18 allocs/op",
            "extra": "18379 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack - ns/op",
            "value": 130486,
            "unit": "ns/op",
            "extra": "18379 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack - MB/s",
            "value": 149.53,
            "unit": "MB/s",
            "extra": "18379 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack - B/op",
            "value": 35197,
            "unit": "B/op",
            "extra": "18379 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "18379 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf",
            "value": 4731,
            "unit": "ns/op\t1773.56 MB/s\t   17524 B/op\t       5 allocs/op",
            "extra": "506962 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf - ns/op",
            "value": 4731,
            "unit": "ns/op",
            "extra": "506962 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf - MB/s",
            "value": 1773.56,
            "unit": "MB/s",
            "extra": "506962 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf - B/op",
            "value": 17524,
            "unit": "B/op",
            "extra": "506962 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeries/metric_1024/decode/qdf - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "506962 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json",
            "value": 134893,
            "unit": "ns/op\t 215.16 MB/s\t   32879 B/op\t       2 allocs/op",
            "extra": "17775 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json - ns/op",
            "value": 134893,
            "unit": "ns/op",
            "extra": "17775 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json - MB/s",
            "value": 215.16,
            "unit": "MB/s",
            "extra": "17775 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json - B/op",
            "value": 32879,
            "unit": "B/op",
            "extra": "17775 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "17775 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack",
            "value": 104777,
            "unit": "ns/op\t 186.29 MB/s\t   65626 B/op\t      12 allocs/op",
            "extra": "22881 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack - ns/op",
            "value": 104777,
            "unit": "ns/op",
            "extra": "22881 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack - MB/s",
            "value": 186.29,
            "unit": "MB/s",
            "extra": "22881 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack - B/op",
            "value": 65626,
            "unit": "B/op",
            "extra": "22881 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/msgpack - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "22881 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf",
            "value": 4238,
            "unit": "ns/op\t1981.66 MB/s\t    9668 B/op\t       3 allocs/op",
            "extra": "554168 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf - ns/op",
            "value": 4238,
            "unit": "ns/op",
            "extra": "554168 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf - MB/s",
            "value": 1981.66,
            "unit": "MB/s",
            "extra": "554168 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf - B/op",
            "value": 9668,
            "unit": "B/op",
            "extra": "554168 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "554168 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json",
            "value": 466132,
            "unit": "ns/op\t  62.27 MB/s\t   54088 B/op\t      40 allocs/op",
            "extra": "5062 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json - ns/op",
            "value": 466132,
            "unit": "ns/op",
            "extra": "5062 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json - MB/s",
            "value": 62.27,
            "unit": "MB/s",
            "extra": "5062 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json - B/op",
            "value": 54088,
            "unit": "B/op",
            "extra": "5062 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/json - allocs/op",
            "value": 40,
            "unit": "allocs/op",
            "extra": "5062 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack",
            "value": 130314,
            "unit": "ns/op\t 149.78 MB/s\t   35205 B/op\t      18 allocs/op",
            "extra": "18408 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack - ns/op",
            "value": 130314,
            "unit": "ns/op",
            "extra": "18408 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack - MB/s",
            "value": 149.78,
            "unit": "MB/s",
            "extra": "18408 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack - B/op",
            "value": 35205,
            "unit": "B/op",
            "extra": "18408 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "18408 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf",
            "value": 4727,
            "unit": "ns/op\t1776.66 MB/s\t   17532 B/op\t       5 allocs/op",
            "extra": "511568 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf - ns/op",
            "value": 4727,
            "unit": "ns/op",
            "extra": "511568 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf - MB/s",
            "value": 1776.66,
            "unit": "MB/s",
            "extra": "511568 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf - B/op",
            "value": 17532,
            "unit": "B/op",
            "extra": "511568 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmooth/metric_smooth_1024/decode/qdf - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "511568 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json",
            "value": 135208,
            "unit": "ns/op\t 214.66 MB/s\t   32880 B/op\t       2 allocs/op",
            "extra": "17743 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json - ns/op",
            "value": 135208,
            "unit": "ns/op",
            "extra": "17743 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json - MB/s",
            "value": 214.66,
            "unit": "MB/s",
            "extra": "17743 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json - B/op",
            "value": 32880,
            "unit": "B/op",
            "extra": "17743 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "17743 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack",
            "value": 105028,
            "unit": "ns/op\t 185.85 MB/s\t   65626 B/op\t      12 allocs/op",
            "extra": "22792 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack - ns/op",
            "value": 105028,
            "unit": "ns/op",
            "extra": "22792 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack - MB/s",
            "value": 185.85,
            "unit": "MB/s",
            "extra": "22792 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack - B/op",
            "value": 65626,
            "unit": "B/op",
            "extra": "22792 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/msgpack - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "22792 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf",
            "value": 49535,
            "unit": "ns/op\t  46.57 MB/s\t   11323 B/op\t      14 allocs/op",
            "extra": "48435 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf - ns/op",
            "value": 49535,
            "unit": "ns/op",
            "extra": "48435 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf - MB/s",
            "value": 46.57,
            "unit": "MB/s",
            "extra": "48435 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf - B/op",
            "value": 11323,
            "unit": "B/op",
            "extra": "48435 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/encode/qdf - allocs/op",
            "value": 14,
            "unit": "allocs/op",
            "extra": "48435 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json",
            "value": 466339,
            "unit": "ns/op\t  62.24 MB/s\t   54088 B/op\t      40 allocs/op",
            "extra": "5007 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json - ns/op",
            "value": 466339,
            "unit": "ns/op",
            "extra": "5007 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json - MB/s",
            "value": 62.24,
            "unit": "MB/s",
            "extra": "5007 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json - B/op",
            "value": 54088,
            "unit": "B/op",
            "extra": "5007 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/json - allocs/op",
            "value": 40,
            "unit": "allocs/op",
            "extra": "5007 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack",
            "value": 130794,
            "unit": "ns/op\t 149.23 MB/s\t   35205 B/op\t      18 allocs/op",
            "extra": "18319 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack - ns/op",
            "value": 130794,
            "unit": "ns/op",
            "extra": "18319 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack - MB/s",
            "value": 149.23,
            "unit": "MB/s",
            "extra": "18319 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack - B/op",
            "value": 35205,
            "unit": "B/op",
            "extra": "18319 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/msgpack - allocs/op",
            "value": 18,
            "unit": "allocs/op",
            "extra": "18319 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf",
            "value": 42940,
            "unit": "ns/op\t  53.73 MB/s\t   17611 B/op\t       7 allocs/op",
            "extra": "55714 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf - ns/op",
            "value": 42940,
            "unit": "ns/op",
            "extra": "55714 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf - MB/s",
            "value": 53.73,
            "unit": "MB/s",
            "extra": "55714 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf - B/op",
            "value": 17611,
            "unit": "B/op",
            "extra": "55714 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_MetricSeriesSmoothCompress/metric_smooth_1024_compress/decode/qdf - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "55714 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json",
            "value": 28712,
            "unit": "ns/op\t 144.05 MB/s\t    4913 B/op\t       2 allocs/op",
            "extra": "83270 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json - ns/op",
            "value": 28712,
            "unit": "ns/op",
            "extra": "83270 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json - MB/s",
            "value": 144.05,
            "unit": "MB/s",
            "extra": "83270 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json - B/op",
            "value": 4913,
            "unit": "B/op",
            "extra": "83270 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "83270 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack",
            "value": 39702,
            "unit": "ns/op\t 233.01 MB/s\t   32805 B/op\t      11 allocs/op",
            "extra": "60494 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack - ns/op",
            "value": 39702,
            "unit": "ns/op",
            "extra": "60494 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack - MB/s",
            "value": 233.01,
            "unit": "MB/s",
            "extra": "60494 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack - B/op",
            "value": 32805,
            "unit": "B/op",
            "extra": "60494 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/msgpack - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "60494 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf",
            "value": 4869,
            "unit": "ns/op\t  20.33 MB/s\t     208 B/op\t       3 allocs/op",
            "extra": "490912 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf - ns/op",
            "value": 4869,
            "unit": "ns/op",
            "extra": "490912 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf - MB/s",
            "value": 20.33,
            "unit": "MB/s",
            "extra": "490912 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf - B/op",
            "value": 208,
            "unit": "B/op",
            "extra": "490912 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "490912 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json",
            "value": 121577,
            "unit": "ns/op\t  34.02 MB/s\t   25504 B/op\t      19 allocs/op",
            "extra": "19776 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json - ns/op",
            "value": 121577,
            "unit": "ns/op",
            "extra": "19776 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json - MB/s",
            "value": 34.02,
            "unit": "MB/s",
            "extra": "19776 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json - B/op",
            "value": 25504,
            "unit": "B/op",
            "extra": "19776 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/json - allocs/op",
            "value": 19,
            "unit": "allocs/op",
            "extra": "19776 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack",
            "value": 50627,
            "unit": "ns/op\t 182.73 MB/s\t   16570 B/op\t       8 allocs/op",
            "extra": "47268 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack - ns/op",
            "value": 50627,
            "unit": "ns/op",
            "extra": "47268 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack - MB/s",
            "value": 182.73,
            "unit": "MB/s",
            "extra": "47268 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack - B/op",
            "value": 16570,
            "unit": "B/op",
            "extra": "47268 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/msgpack - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "47268 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf",
            "value": 2523,
            "unit": "ns/op\t  39.24 MB/s\t    8258 B/op\t       3 allocs/op",
            "extra": "947937 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf - ns/op",
            "value": 2523,
            "unit": "ns/op",
            "extra": "947937 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf - MB/s",
            "value": 39.24,
            "unit": "MB/s",
            "extra": "947937 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf - B/op",
            "value": 8258,
            "unit": "B/op",
            "extra": "947937 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_StatusBatch/status_1024/decode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "947937 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json",
            "value": 60627,
            "unit": "ns/op\t 138.30 MB/s\t    9522 B/op\t       2 allocs/op",
            "extra": "39432 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json - ns/op",
            "value": 60627,
            "unit": "ns/op",
            "extra": "39432 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json - MB/s",
            "value": 138.3,
            "unit": "MB/s",
            "extra": "39432 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json - B/op",
            "value": 9522,
            "unit": "B/op",
            "extra": "39432 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "39432 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack",
            "value": 24909,
            "unit": "ns/op\t 155.13 MB/s\t    8225 B/op\t       9 allocs/op",
            "extra": "96250 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack - ns/op",
            "value": 24909,
            "unit": "ns/op",
            "extra": "96250 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack - MB/s",
            "value": 155.13,
            "unit": "MB/s",
            "extra": "96250 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack - B/op",
            "value": 8225,
            "unit": "B/op",
            "extra": "96250 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/msgpack - allocs/op",
            "value": 9,
            "unit": "allocs/op",
            "extra": "96250 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf",
            "value": 928.6,
            "unit": "ns/op\t3341.46 MB/s\t    3297 B/op\t       3 allocs/op",
            "extra": "2594936 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf - ns/op",
            "value": 928.6,
            "unit": "ns/op",
            "extra": "2594936 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf - MB/s",
            "value": 3341.46,
            "unit": "MB/s",
            "extra": "2594936 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf - B/op",
            "value": 3297,
            "unit": "B/op",
            "extra": "2594936 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "2594936 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json",
            "value": 161994,
            "unit": "ns/op\t  51.76 MB/s\t    7832 B/op\t      17 allocs/op",
            "extra": "14806 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json - ns/op",
            "value": 161994,
            "unit": "ns/op",
            "extra": "14806 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json - MB/s",
            "value": 51.76,
            "unit": "MB/s",
            "extra": "14806 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json - B/op",
            "value": 7832,
            "unit": "B/op",
            "extra": "14806 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/json - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "14806 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack",
            "value": 35425,
            "unit": "ns/op\t 109.08 MB/s\t    6320 B/op\t       8 allocs/op",
            "extra": "66962 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack - ns/op",
            "value": 35425,
            "unit": "ns/op",
            "extra": "66962 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack - MB/s",
            "value": 109.08,
            "unit": "MB/s",
            "extra": "66962 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack - B/op",
            "value": 6320,
            "unit": "B/op",
            "extra": "66962 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/msgpack - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "66962 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf",
            "value": 748.5,
            "unit": "ns/op\t4145.52 MB/s\t    3129 B/op\t       3 allocs/op",
            "extra": "3207856 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf - ns/op",
            "value": 748.5,
            "unit": "ns/op",
            "extra": "3207856 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf - MB/s",
            "value": 4145.52,
            "unit": "MB/s",
            "extra": "3207856 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf - B/op",
            "value": 3129,
            "unit": "B/op",
            "extra": "3207856 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_EmbeddingVec/embed_768/decode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "3207856 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json",
            "value": 1980,
            "unit": "ns/op\t 126.25 MB/s\t     936 B/op\t      22 allocs/op",
            "extra": "1212681 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json - ns/op",
            "value": 1980,
            "unit": "ns/op",
            "extra": "1212681 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json - MB/s",
            "value": 126.25,
            "unit": "MB/s",
            "extra": "1212681 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json - B/op",
            "value": 936,
            "unit": "B/op",
            "extra": "1212681 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/json - allocs/op",
            "value": 22,
            "unit": "allocs/op",
            "extra": "1212681 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack",
            "value": 1634,
            "unit": "ns/op\t 120.58 MB/s\t     680 B/op\t      15 allocs/op",
            "extra": "1460740 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack - ns/op",
            "value": 1634,
            "unit": "ns/op",
            "extra": "1460740 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack - MB/s",
            "value": 120.58,
            "unit": "MB/s",
            "extra": "1460740 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack - B/op",
            "value": 680,
            "unit": "B/op",
            "extra": "1460740 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/msgpack - allocs/op",
            "value": 15,
            "unit": "allocs/op",
            "extra": "1460740 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf",
            "value": 1244,
            "unit": "ns/op\t 180.83 MB/s\t     368 B/op\t       3 allocs/op",
            "extra": "1925668 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf - ns/op",
            "value": 1244,
            "unit": "ns/op",
            "extra": "1925668 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf - MB/s",
            "value": 180.83,
            "unit": "MB/s",
            "extra": "1925668 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf - B/op",
            "value": 368,
            "unit": "B/op",
            "extra": "1925668 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/encode/qdf - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1925668 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json",
            "value": 5761,
            "unit": "ns/op\t  43.40 MB/s\t    1352 B/op\t      41 allocs/op",
            "extra": "417704 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json - ns/op",
            "value": 5761,
            "unit": "ns/op",
            "extra": "417704 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json - MB/s",
            "value": 43.4,
            "unit": "MB/s",
            "extra": "417704 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json - B/op",
            "value": 1352,
            "unit": "B/op",
            "extra": "417704 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/json - allocs/op",
            "value": 41,
            "unit": "allocs/op",
            "extra": "417704 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack",
            "value": 2681,
            "unit": "ns/op\t  73.49 MB/s\t    1064 B/op\t      34 allocs/op",
            "extra": "843715 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack - ns/op",
            "value": 2681,
            "unit": "ns/op",
            "extra": "843715 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack - MB/s",
            "value": 73.49,
            "unit": "MB/s",
            "extra": "843715 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack - B/op",
            "value": 1064,
            "unit": "B/op",
            "extra": "843715 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/msgpack - allocs/op",
            "value": 34,
            "unit": "allocs/op",
            "extra": "843715 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf",
            "value": 1560,
            "unit": "ns/op\t 144.25 MB/s\t     891 B/op\t      16 allocs/op",
            "extra": "1533924 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf - ns/op",
            "value": 1560,
            "unit": "ns/op",
            "extra": "1533924 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf - MB/s",
            "value": 144.25,
            "unit": "MB/s",
            "extra": "1533924 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf - B/op",
            "value": 891,
            "unit": "B/op",
            "extra": "1533924 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Config/config/decode/qdf - allocs/op",
            "value": 16,
            "unit": "allocs/op",
            "extra": "1533924 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json",
            "value": 1826647,
            "unit": "ns/op\t 391.32 MB/s\t  730804 B/op\t       3 allocs/op",
            "extra": "1304 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json - ns/op",
            "value": 1826647,
            "unit": "ns/op",
            "extra": "1304 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json - MB/s",
            "value": 391.32,
            "unit": "MB/s",
            "extra": "1304 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json - B/op",
            "value": 730804,
            "unit": "B/op",
            "extra": "1304 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/json - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "1304 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack",
            "value": 2508585,
            "unit": "ns/op\t 222.64 MB/s\t 2217450 B/op\t    5018 allocs/op",
            "extra": "922 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack - ns/op",
            "value": 2508585,
            "unit": "ns/op",
            "extra": "922 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack - MB/s",
            "value": 222.64,
            "unit": "MB/s",
            "extra": "922 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack - B/op",
            "value": 2217450,
            "unit": "B/op",
            "extra": "922 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/msgpack - allocs/op",
            "value": 5018,
            "unit": "allocs/op",
            "extra": "922 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf",
            "value": 1808501,
            "unit": "ns/op\t 106.30 MB/s\t 1518381 B/op\t      51 allocs/op",
            "extra": "1306 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf - ns/op",
            "value": 1808501,
            "unit": "ns/op",
            "extra": "1306 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf - MB/s",
            "value": 106.3,
            "unit": "MB/s",
            "extra": "1306 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf - B/op",
            "value": 1518381,
            "unit": "B/op",
            "extra": "1306 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/encode/qdf - allocs/op",
            "value": 51,
            "unit": "allocs/op",
            "extra": "1306 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json",
            "value": 13167383,
            "unit": "ns/op\t  54.29 MB/s\t 3074431 B/op\t   45025 allocs/op",
            "extra": "181 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json - ns/op",
            "value": 13167383,
            "unit": "ns/op",
            "extra": "181 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json - MB/s",
            "value": 54.29,
            "unit": "MB/s",
            "extra": "181 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json - B/op",
            "value": 3074431,
            "unit": "B/op",
            "extra": "181 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/json - allocs/op",
            "value": 45025,
            "unit": "allocs/op",
            "extra": "181 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack",
            "value": 4822497,
            "unit": "ns/op\t 115.81 MB/s\t 1602199 B/op\t   40008 allocs/op",
            "extra": "493 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack - ns/op",
            "value": 4822497,
            "unit": "ns/op",
            "extra": "493 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack - MB/s",
            "value": 115.81,
            "unit": "MB/s",
            "extra": "493 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack - B/op",
            "value": 1602199,
            "unit": "B/op",
            "extra": "493 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/msgpack - allocs/op",
            "value": 40008,
            "unit": "allocs/op",
            "extra": "493 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf",
            "value": 2405030,
            "unit": "ns/op\t  79.93 MB/s\t 1822941 B/op\t   16296 allocs/op",
            "extra": "992 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf - ns/op",
            "value": 2405030,
            "unit": "ns/op",
            "extra": "992 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf - MB/s",
            "value": 79.93,
            "unit": "MB/s",
            "extra": "992 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf - B/op",
            "value": 1822941,
            "unit": "B/op",
            "extra": "992 times\n4 procs"
          },
          {
            "name": "BenchmarkProfile_Archive/archive_5k/decode/qdf - allocs/op",
            "value": 16296,
            "unit": "allocs/op",
            "extra": "992 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/json",
            "value": 53671,
            "unit": "ns/op\t   10979 B/op\t       2 allocs/op",
            "extra": "44587 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/json - ns/op",
            "value": 53671,
            "unit": "ns/op",
            "extra": "44587 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/json - B/op",
            "value": 10979,
            "unit": "B/op",
            "extra": "44587 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/json - allocs/op",
            "value": 2,
            "unit": "allocs/op",
            "extra": "44587 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/msgpack",
            "value": 57985,
            "unit": "ns/op\t   32852 B/op\t      11 allocs/op",
            "extra": "41614 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/msgpack - ns/op",
            "value": 57985,
            "unit": "ns/op",
            "extra": "41614 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/msgpack - B/op",
            "value": 32852,
            "unit": "B/op",
            "extra": "41614 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/msgpack - allocs/op",
            "value": 11,
            "unit": "allocs/op",
            "extra": "41614 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_fast",
            "value": 8569,
            "unit": "ns/op\t    6978 B/op\t       3 allocs/op",
            "extra": "312240 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_fast - ns/op",
            "value": 8569,
            "unit": "ns/op",
            "extra": "312240 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_fast - B/op",
            "value": 6978,
            "unit": "B/op",
            "extra": "312240 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_fast - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "312240 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_qpack",
            "value": 2846,
            "unit": "ns/op\t    2496 B/op\t       3 allocs/op",
            "extra": "861309 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_qpack - ns/op",
            "value": 2846,
            "unit": "ns/op",
            "extra": "861309 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_qpack - B/op",
            "value": 2496,
            "unit": "B/op",
            "extra": "861309 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_qpack - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "861309 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_dense",
            "value": 2988,
            "unit": "ns/op\t    2497 B/op\t       3 allocs/op",
            "extra": "808988 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_dense - ns/op",
            "value": 2988,
            "unit": "ns/op",
            "extra": "808988 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_dense - B/op",
            "value": 2497,
            "unit": "B/op",
            "extra": "808988 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Encode/qdf_dense - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "808988 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/json",
            "value": 209545,
            "unit": "ns/op\t   21288 B/op\t      41 allocs/op",
            "extra": "10000 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/json - ns/op",
            "value": 209545,
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
            "value": 73311,
            "unit": "ns/op\t   21427 B/op\t      22 allocs/op",
            "extra": "32743 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/msgpack - ns/op",
            "value": 73311,
            "unit": "ns/op",
            "extra": "32743 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/msgpack - B/op",
            "value": 21427,
            "unit": "B/op",
            "extra": "32743 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/msgpack - allocs/op",
            "value": 22,
            "unit": "allocs/op",
            "extra": "32743 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_fast",
            "value": 13951,
            "unit": "ns/op\t   10594 B/op\t       5 allocs/op",
            "extra": "171822 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_fast - ns/op",
            "value": 13951,
            "unit": "ns/op",
            "extra": "171822 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_fast - B/op",
            "value": 10594,
            "unit": "B/op",
            "extra": "171822 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_fast - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "171822 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_qpack",
            "value": 3341,
            "unit": "ns/op\t   10595 B/op\t       5 allocs/op",
            "extra": "709242 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_qpack - ns/op",
            "value": 3341,
            "unit": "ns/op",
            "extra": "709242 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_qpack - B/op",
            "value": 10595,
            "unit": "B/op",
            "extra": "709242 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_qpack - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "709242 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_dense",
            "value": 3638,
            "unit": "ns/op\t   10676 B/op\t       7 allocs/op",
            "extra": "647366 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_dense - ns/op",
            "value": 3638,
            "unit": "ns/op",
            "extra": "647366 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_dense - B/op",
            "value": 10676,
            "unit": "B/op",
            "extra": "647366 times\n4 procs"
          },
          {
            "name": "BenchmarkQPack_Decode/qdf_dense - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "647366 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/json",
            "value": 3694,
            "unit": "ns/op\t     488 B/op\t      12 allocs/op",
            "extra": "636081 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/json - ns/op",
            "value": 3694,
            "unit": "ns/op",
            "extra": "636081 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/json - B/op",
            "value": 488,
            "unit": "B/op",
            "extra": "636081 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/json - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "636081 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/msgpack",
            "value": 1189,
            "unit": "ns/op\t     312 B/op\t       9 allocs/op",
            "extra": "2018458 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/msgpack - ns/op",
            "value": 1189,
            "unit": "ns/op",
            "extra": "2018458 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/msgpack - B/op",
            "value": 312,
            "unit": "B/op",
            "extra": "2018458 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/msgpack - allocs/op",
            "value": 9,
            "unit": "allocs/op",
            "extra": "2018458 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/qdf_fast",
            "value": 533.7,
            "unit": "ns/op\t     264 B/op\t       8 allocs/op",
            "extra": "4492531 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/qdf_fast - ns/op",
            "value": 533.7,
            "unit": "ns/op",
            "extra": "4492531 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/qdf_fast - B/op",
            "value": 264,
            "unit": "B/op",
            "extra": "4492531 times\n4 procs"
          },
          {
            "name": "BenchmarkDecode_UniqueLog/qdf_fast - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "4492531 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/json",
            "value": 1657,
            "unit": "ns/op\t     488 B/op\t      12 allocs/op",
            "extra": "1451998 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/json - ns/op",
            "value": 1657,
            "unit": "ns/op",
            "extra": "1451998 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/json - B/op",
            "value": 488,
            "unit": "B/op",
            "extra": "1451998 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/json - allocs/op",
            "value": 12,
            "unit": "allocs/op",
            "extra": "1451998 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/msgpack",
            "value": 559.3,
            "unit": "ns/op\t     312 B/op\t       9 allocs/op",
            "extra": "4306875 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/msgpack - ns/op",
            "value": 559.3,
            "unit": "ns/op",
            "extra": "4306875 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/msgpack - B/op",
            "value": 312,
            "unit": "B/op",
            "extra": "4306875 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/msgpack - allocs/op",
            "value": 9,
            "unit": "allocs/op",
            "extra": "4306875 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/qdf_fast",
            "value": 276.3,
            "unit": "ns/op\t     264 B/op\t       8 allocs/op",
            "extra": "8667536 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/qdf_fast - ns/op",
            "value": 276.3,
            "unit": "ns/op",
            "extra": "8667536 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/qdf_fast - B/op",
            "value": 264,
            "unit": "B/op",
            "extra": "8667536 times\n4 procs"
          },
          {
            "name": "BenchmarkDecodeParallel_UniqueLog/qdf_fast - allocs/op",
            "value": 8,
            "unit": "allocs/op",
            "extra": "8667536 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense",
            "value": 388147,
            "unit": "ns/op\t 478.40 MB/s\t  376006 B/op\t    2011 allocs/op",
            "extra": "6165 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense - ns/op",
            "value": 388147,
            "unit": "ns/op",
            "extra": "6165 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense - MB/s",
            "value": 478.4,
            "unit": "MB/s",
            "extra": "6165 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense - B/op",
            "value": 376006,
            "unit": "B/op",
            "extra": "6165 times\n4 procs"
          },
          {
            "name": "BenchmarkStream_LogBatch1k_Dense/encode_stream_dense - allocs/op",
            "value": 2011,
            "unit": "allocs/op",
            "extra": "6165 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/json",
            "value": 2246,
            "unit": "ns/op\t     800 B/op\t      14 allocs/op",
            "extra": "1000000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/json - ns/op",
            "value": 2246,
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
            "value": 1919,
            "unit": "ns/op\t    1008 B/op\t      17 allocs/op",
            "extra": "1251559 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/msgpack - ns/op",
            "value": 1919,
            "unit": "ns/op",
            "extra": "1251559 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/msgpack - B/op",
            "value": 1008,
            "unit": "B/op",
            "extra": "1251559 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/msgpack - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "1251559 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/qdf_fast",
            "value": 1579,
            "unit": "ns/op\t     697 B/op\t      13 allocs/op",
            "extra": "1516050 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/qdf_fast - ns/op",
            "value": 1579,
            "unit": "ns/op",
            "extra": "1516050 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/qdf_fast - B/op",
            "value": 697,
            "unit": "B/op",
            "extra": "1516050 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_UniqueLog/qdf_fast - allocs/op",
            "value": 13,
            "unit": "allocs/op",
            "extra": "1516050 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/json",
            "value": 1041,
            "unit": "ns/op\t     364 B/op\t       5 allocs/op",
            "extra": "2302666 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/json - ns/op",
            "value": 1041,
            "unit": "ns/op",
            "extra": "2302666 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/json - B/op",
            "value": 364,
            "unit": "B/op",
            "extra": "2302666 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/json - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "2302666 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/msgpack",
            "value": 1038,
            "unit": "ns/op\t     538 B/op\t       7 allocs/op",
            "extra": "2310164 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/msgpack - ns/op",
            "value": 1038,
            "unit": "ns/op",
            "extra": "2310164 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/msgpack - B/op",
            "value": 538,
            "unit": "B/op",
            "extra": "2310164 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/msgpack - allocs/op",
            "value": 7,
            "unit": "allocs/op",
            "extra": "2310164 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/qdf_fast",
            "value": 795.9,
            "unit": "ns/op\t     388 B/op\t       5 allocs/op",
            "extra": "3022996 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/qdf_fast - ns/op",
            "value": 795.9,
            "unit": "ns/op",
            "extra": "3022996 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/qdf_fast - B/op",
            "value": 388,
            "unit": "B/op",
            "extra": "3022996 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_MixedTypes/qdf_fast - allocs/op",
            "value": 5,
            "unit": "allocs/op",
            "extra": "3022996 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/json",
            "value": 276340,
            "unit": "ns/op\t  121481 B/op\t       3 allocs/op",
            "extra": "8000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/json - ns/op",
            "value": 276340,
            "unit": "ns/op",
            "extra": "8000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/json - B/op",
            "value": 121481,
            "unit": "B/op",
            "extra": "8000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/json - allocs/op",
            "value": 3,
            "unit": "allocs/op",
            "extra": "8000 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/msgpack",
            "value": 239478,
            "unit": "ns/op\t  190678 B/op\t      10 allocs/op",
            "extra": "9969 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/msgpack - ns/op",
            "value": 239478,
            "unit": "ns/op",
            "extra": "9969 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/msgpack - B/op",
            "value": 190678,
            "unit": "B/op",
            "extra": "9969 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/msgpack - allocs/op",
            "value": 10,
            "unit": "allocs/op",
            "extra": "9969 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/qdf_fast",
            "value": 92802,
            "unit": "ns/op\t   91099 B/op\t       4 allocs/op",
            "extra": "25758 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/qdf_fast - ns/op",
            "value": 92802,
            "unit": "ns/op",
            "extra": "25758 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/qdf_fast - B/op",
            "value": 91099,
            "unit": "B/op",
            "extra": "25758 times\n4 procs"
          },
          {
            "name": "BenchmarkEncode_RandomSize/qdf_fast - allocs/op",
            "value": 4,
            "unit": "allocs/op",
            "extra": "25758 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/json",
            "value": 1110,
            "unit": "ns/op\t     800 B/op\t      14 allocs/op",
            "extra": "2160517 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/json - ns/op",
            "value": 1110,
            "unit": "ns/op",
            "extra": "2160517 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/json - B/op",
            "value": 800,
            "unit": "B/op",
            "extra": "2160517 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/json - allocs/op",
            "value": 14,
            "unit": "allocs/op",
            "extra": "2160517 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/msgpack",
            "value": 1017,
            "unit": "ns/op\t    1008 B/op\t      17 allocs/op",
            "extra": "2383539 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/msgpack - ns/op",
            "value": 1017,
            "unit": "ns/op",
            "extra": "2383539 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/msgpack - B/op",
            "value": 1008,
            "unit": "B/op",
            "extra": "2383539 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/msgpack - allocs/op",
            "value": 17,
            "unit": "allocs/op",
            "extra": "2383539 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/qdf_fast",
            "value": 804.2,
            "unit": "ns/op\t     697 B/op\t      13 allocs/op",
            "extra": "3011504 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/qdf_fast - ns/op",
            "value": 804.2,
            "unit": "ns/op",
            "extra": "3011504 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/qdf_fast - B/op",
            "value": 697,
            "unit": "B/op",
            "extra": "3011504 times\n4 procs"
          },
          {
            "name": "BenchmarkEncodeParallel_UniqueLog/qdf_fast - allocs/op",
            "value": 13,
            "unit": "allocs/op",
            "extra": "3011504 times\n4 procs"
          }
        ]
      }
    ]
  }
}
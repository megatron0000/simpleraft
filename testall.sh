# Tests in order from low-level to high-level
# -v for "verbose": enables print-to-console (log.Printf etc.) inside the tests
go test -v ./storage -failfast -count 2
go test -v ./status -failfast -count 2
go test -v ./raftlog -failfast -count 2
go test -v ./transport -failfast -count 1 # count=1 to avoid issues with simultaneous servers
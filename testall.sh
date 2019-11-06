# Tests in order from low-level to high-level
go test ./storage -failfast -count 2
go test ./status -failfast -count 2
go test ./raftlog -failfast -count 2
go test ./transport -failfast -count 1 # count=1 to avoid issues with simultaneous servers
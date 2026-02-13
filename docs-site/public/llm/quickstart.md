# Wrkr Quickstart

```bash
git clone https://github.com/davidahmann/wrkr.git
cd wrkr
make build
./bin/wrkr demo --json
./bin/wrkr verify <job_id> --json
```

Then run a real job:

```bash
./bin/wrkr init jobspec.yaml
./bin/wrkr submit jobspec.yaml --job-id job_demo --json
./bin/wrkr checkpoint list job_demo --json
```

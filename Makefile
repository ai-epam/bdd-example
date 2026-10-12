.PHONY: dev backend frontend bdd test tidy clean
# TAGS is passed to godog, e.g. make bdd TAGS=@smoke or TAGS='~@wip'
export TAGS

dev:            ## run backend + frontend concurrently
	@(trap 'kill 0' INT; $(MAKE) backend & $(MAKE) frontend & wait)

backend:
	cd backend && go run ./cmd/server

frontend:
	cd frontend && npm install --silent && npm run dev

bdd:            ## run the full godog suite (includes the failing @wip demo) and write bdd-report.json
	cd backend && go test . -run TestFeatures -count=1 -v

test:           ## everything except the deliberately failing @wip scenarios
	cd backend && TAGS='~@wip' go test ./... -count=1

tidy:
	cd backend && go mod tidy

clean:
	rm -f bdd-report.json

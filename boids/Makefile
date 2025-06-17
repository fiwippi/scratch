.PHONY: dev
dev:
	mkdir -p dev
	wasm-pack build --target web --no-pack --no-typescript ./boids
	cp ./boids/pkg/lib_boids.js ./dev/lib_boids.js
	cp ./boids/pkg/lib_boids_bg.wasm ./dev/lib_boids_bg.wasm
	cp ./www/assets/index.html ./dev/index.html
	python3 -m http.server -d ./dev 3000

build:
	wasm-pack build --target web --no-pack --no-typescript ./boids
	cargo build -p www

run:
	cargo run

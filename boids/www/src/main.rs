use std::include_bytes;
use std::net::SocketAddr;

use axum::response::IntoResponse;
use tracing::Level;
use tower_http::trace::{TraceLayer, DefaultMakeSpan, DefaultOnResponse, DefaultOnFailure};
use time::macros::format_description;
use tracing_subscriber::fmt::time::UtcTime;
use axum::{response::Html, routing::get, Router};
use http::{header, HeaderMap};

// Server

#[tokio::main]
async fn main() {
    // Setup tracing
    let timer = UtcTime::new(format_description!(
            "[day]-[month]-[year] [hour]:[minute]:[second].[subsecond]"));
    tracing_subscriber::fmt()
        .with_timer(timer)
        .compact()
        .init();

    // Build our application with a route
    let app = Router::new()
        .route("/", get(index))
        .route("/lib_boids.js", get(js))
        .route("/lib_boids_bg.wasm", get(wasm))
        .layer(TraceLayer::new_for_http()
            .make_span_with(DefaultMakeSpan::new().level(Level::INFO))
            .on_response(DefaultOnResponse::new().level(Level::INFO))
            .on_failure(DefaultOnFailure::new().level(Level::INFO))
            );

    // Build the server
    let addr = SocketAddr::from(([0, 0, 0, 0], 3000));
    tracing::info!("listening on {}", addr);
    let server = hyper::Server::bind(&addr)
        .serve(app.into_make_service());

    // Run the server
    if let Err(e) = server.await {
        eprintln!("server error: {}", e);
    }
}

// Routes

async fn index() -> Html<&'static [u8]> {
    Html(include_bytes!("../assets/index.html"))
}

async fn js() -> impl IntoResponse { 
    let mut headers = HeaderMap::new();
    headers.insert(header::CONTENT_TYPE, "text/javascript; charset=utf-8".parse().unwrap());
 
    (headers, include_bytes!("../../boids/pkg/lib_boids.js"))
}

async fn wasm() -> impl IntoResponse { 
    let mut headers = HeaderMap::new();
    headers.insert(header::CONTENT_TYPE, "application/wasm".parse().unwrap());
 
    (headers, include_bytes!("../../boids/pkg/lib_boids_bg.wasm"))
}

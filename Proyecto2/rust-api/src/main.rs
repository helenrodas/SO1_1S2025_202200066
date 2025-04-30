use actix_web::{post, web, App, HttpResponse, HttpServer, Responder};
use serde::{Deserialize, Serialize};
use std::io;
use log::info;

#[derive(Deserialize, Serialize, Clone, Debug)]
struct Tweet {
    description: String,
    country: String,
    weather: String,
}

#[post("/input")]
async fn handle_tweet(data: web::Json<Tweet>) -> impl Responder {
    info!("Recibido en /input: {:?}", data);
    info!("Datos del tweet:");
    info!("Descripción: {}", data.description);
    info!("País: {}", data.country);
    info!("Clima: {}", data.weather);

    HttpResponse::Ok().json("Tweet recibido e impreso en consola")
}

#[actix_web::main]
async fn main() -> io::Result<()> {
    // Inicializar logger
    env_logger::init_from_env(env_logger::Env::new().default_filter_or("info"));
    
    info!("Iniciando servidor Rust en 0.0.0.0:8080");
    
    HttpServer::new(|| {
        App::new()
            .service(handle_tweet)
    })
    .bind("0.0.0.0:8080")?
    .run()
    .await
}
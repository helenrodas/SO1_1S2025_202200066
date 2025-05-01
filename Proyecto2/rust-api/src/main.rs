use actix_web::{post, web, App, HttpResponse, HttpServer, Responder};
use serde::{Deserialize, Serialize};
use std::io;
use log::info;
use reqwest::Client;

#[derive(Deserialize, Serialize, Clone, Debug)]
struct Tweet {
    description: String,
    country: String,
    weather: String,
}

#[post("/input")]
async fn handle_tweet(data: web::Json<Tweet>, client: web::Data<Client>) -> impl Responder {
    // Imprimir logs como antes
    info!("Recibido en /input: {:?}", data);
    info!("Datos del tweet:");
    info!("Descripción: {}", data.description);
    info!("País: {}", data.country);
    info!("Clima: {}", data.weather);

    // URL del servicio go-rest dentro del clúster de Kubernetes
    let go_url = "http://go-rest.tweets-clima.svc.cluster.local:8080/input";

    // Enviar solicitud POST a go-rest
    match client.post(go_url).json(&data).send().await {
        Ok(response) => {
            info!("Datos enviados a Go: {:?}", data);
            info!("Respuesta de Go: {:?}", response.status());
            if response.status().is_success() {
                // Si la solicitud a go-rest fue exitosa, devolver la respuesta original
                HttpResponse::Ok().json("Tweet recibido e impreso en consola")
            } else {
                // Si go-rest devuelve un error, loguearlo y devolver un error
                info!("Error al enviar datos a Go: {:?}", response.status());
                HttpResponse::InternalServerError().json("Error al procesar el tweet en go-rest")
            }
        }
        Err(err) => {
            // Si falla la conexión a go-rest, loguear el error y devolver un error
            info!("Error al conectar con go-rest: {:?}", err);
            HttpResponse::InternalServerError().json("Error al conectar con go-rest")
        }
    }
}

#[actix_web::main]
async fn main() -> io::Result<()> {
    // Inicializar logger
    env_logger::init_from_env(env_logger::Env::new().default_filter_or("info"));
    
    // Crear cliente HTTP
    let client = Client::new();
    
    info!("Iniciando servidor Rust en 0.0.0.0:8080");
    
    HttpServer::new(move || {
        App::new()
            .app_data(web::Data::new(client.clone())) // Pasar el cliente HTTP a los handlers
            .service(handle_tweet)
    })
    .bind("0.0.0.0:8080")?
    .run()
    .await
}
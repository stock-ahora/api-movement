<<<<<<< HEAD
API Movimiento - TrueStock движения
Sistema de procesamiento y consulta de movimientos de inventario, que consume eventos desde RabbitMQ y expone una API REST para la trazabilidad de los mismos.

🚀 Características Principales
🔄 Consumidor Asíncrono: Procesa eventos de movimientos de inventario de forma robusta y desacoplada utilizando RabbitMQ.

🌐 API REST de Consulta: Endpoints para consultar el historial de movimientos registrados en el sistema.

🐘 Base de Datos Relacional: Almacena de forma persistente todos los movimientos en PostgreSQL.

🐳 Infraestructura como Código (IaC): Preparado para ser gestionado y desplegado utilizando Docker y Makefiles.

⚙️ Configuración Centralizada: Gestión de la configuración a través de variables de entorno para una fácil adaptación entre entornos.

🏗️ Estructura del Proyecto
El proyecto está organizado con un enfoque en la separación de responsabilidades:

Bash

api-movement/
├── 🚀 main.go                     # Punto de entrada: inicia el consumidor y el servidor API
├── 📦 go.mod                      # Módulo y dependencias de Go
├── 🔑 .env                        # Variables de entorno locales (NO incluir en Git)
├── 🐳 Dockerfile                  # (Pendiente) Definición de la imagen Docker
├── 🛠️ Makefile                    # (Pendiente) Comandos de automatización
│
├── api/
│   ├── 📡 handler.go              # Manejadores de las peticiones HTTP
│   └── 🗺️ router.go               # Definición de las rutas de la API
│
├── config/
│   └── 📜 secret_manager.go       # Carga de configuración desde .env
│
├── consumer/
│   └── 🐇 rabbitmq_consumer.go    # Lógica del consumidor de RabbitMQ
│
├── database/
│   └── 🐘 database.go             # Conexión a la base de datos PostgreSQL
│
├── models/
│   └── 📝 models.go               # Structs y modelos de datos de la aplicación
│
└── services/
└── ✨ movimiento_service.go   # Lógica de negocio para procesar y consultar movimientos
⚙️ Configuración Inicial
Sigue estos pasos para poner en marcha el proyecto.

1. Requisitos Previos
   ✅ Go (versión 1.18 o superior)

✅ Acceso a una base de datos PostgreSQL

✅ Acceso a un broker RabbitMQ

2. Clonar y Preparar
   Bash

# Clonar el repositorio
git clone <tu-repo>
cd api-movement

# Instalar dependencias
go mod tidy
3. Configurar Variables de Entorno
   Crea un archivo .env en la raíz del proyecto. ⚠️ ¡IMPORTANTE! Asegúrate de que este archivo esté en tu .gitignore y nunca lo subas al repositorio.

Bash

# .env - Archivo de ejemplo
# === PostgreSQL Configuration ===
DB_USER=db_usuarios
DB_PASSWORD=tu_password_secreto
DB_HOST=tu_host_rds.us-east-2.rds.amazonaws.com
DB_PORT=54320
DB_NAME=appdb

# === RabbitMQ Configuration ===
RABBIT_USER=adminuser
RABBIT_PASSWORD=tu_password_secreto
RABBIT_HOST=tu_broker_mq.us-east-2.on.aws
RABBIT_PORT=567000
RABBIT_VHOST=/

▶️ Ejecución Local
Una vez configurado, ejecuta la aplicación con el siguiente comando:

Bash

go run main.go
Si todo es correcto, verás un log de inicio similar a este:

Fragmento de código

2025/10/13 16:06:03 === Iniciando API y Consumidor de Movimientos ===
2025/10/13 16:06:03 Cargando configuración desde .env
2025/10/13 16:06:05 ✅ Conectado a PostgreSQL
2025/10/13 16:06:05 ✅ Servicio de movimientos listo
2025/10/13 16:06:05 🚀 Servidor HTTP escuchando en el puerto :8080
2025/10/13 16:06:06 ✅ Conectado a RabbitMQ
2025/10/13 16:06:06 ✅ Esperando mensajes en queue 'movement.generated'...
📨 Estructura del Mensaje RabbitMQ
El consumidor espera mensajes en la cola movement.generated con la siguiente estructura JSON:

JSON

{
"id": "a1b2c3d4-e5f6-7890-1234-567890abcdef",
"request_id": "f0e9d8c7-b6a5-4321-fedc-ba9876543210",
"products": [
{
"product_id": "c1d2e3f4-a5b6-c7d8-e9f0-1234567890ab",
"count": 10,
"movement_id": "d1e2f3a4-b5c6-d7e8-f9a0-b1c2d3e4f5a6",
"date_limit": "2025-12-31T23:59:59Z",
"movement_type": 1,
"created_at": "2025-10-13T16:00:00Z"
}
]
}
🌐 Endpoints Disponibles
La API expone los siguientes endpoints para consulta:

Consultar todos los movimientos
Retorna una lista de los últimos 100 movimientos registrados.

<pre><code><span style="color: #61AFEF;">GET</span> /movements</code></pre>

Consultar un movimiento por ID
Retorna los detalles de un movimiento específico, incluyendo la lista de productos asociados.

<pre><code><span style="color: #61AFEF;">GET</span> /movements/{id}</code></pre>

{id}: El UUID del movimiento a consultar.

🔄 Flujo de Procesamiento de Mensajes
📥 Recepción del Mensaje: El consumidor escucha en la cola movement.generated.

🧾 Decodificación: El cuerpo del mensaje (JSON) se mapea al struct MovementsEvent.

⏳ Transacción en Base de Datos: Se inicia una transacción en PostgreSQL para garantizar la atomicidad.

💾 Persistencia:

Se inserta un registro principal en la tabla movement.

Se itera sobre los productos y se inserta cada uno en request_per_product.

✅ Confirmación: Si la transacción es exitosa, se envía un ACK a RabbitMQ para eliminar el mensaje. Si falla, se envía un NACK para que el mensaje sea reintentado.

🗺️ Futuras Mejoras y Próximos Pasos
Este proyecto tiene una base sólida. Las próximas mejoras planeadas incluyen:

[ ] 🐳 Contenerización: Finalizar el Dockerfile y docker-compose.yml para facilitar el despliegue.

[ ] 🤖 Automatización: Implementar los comandos del Makefile para agilizar tareas (build, test, run).

[ ] 🛡️ Seguridad Avanzada:

Implementar encriptación (ej. AES-256) para datos sensibles en la base de datos.

Integrar un sistema de autenticación (ej. JWT) para los endpoints de la API.

[ ] 🔍 API de Consulta Avanzada: Expandir los endpoints para permitir filtrado por producto, tipo de movimiento, fechas, etc.

[ ] 📊 Monitoreo y Métricas: Añadir un endpoint /metrics para Prometheus y un endpoint /health detallado.

[ ] 🔗 Integración con API Stock: Desarrollar la lógica para consultar y actualizar el stock en el servicio api-stock.
=======
# api-movement
>>>>>>> ed5b1d630508ad28ba1b28bfaf42acd9af87d403

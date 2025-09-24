# api-movement
Resumen funcional (lo que debe hacer api movimiento)

1.Exponer endpoint HTTP (POST) para recibir movimientos (entrada/salida) desde tu API stock / OCR / frontend.
2.Validar, normalizar y encolar el evento en RabbitMQ (trazabilidad inmediata).
3.Worker (consumidor) procesa mensajes de la cola, calcula stock, registra movimiento en DB (Postgres) y guarda registro encriptado en la tabla movimiento.
4.Guardar metadata para trazabilidad: cantidad, timestamp (hora), producto_id, origen (usuario/sistema), correlación, estado, hasho y/o firmas .
5.Usar Secret Manager para recuperar secretos (clave de cifrado, credenciales DB, credenciales RabbitMQ).
6.Garantizar idempotencia y logs.

idempotencia=Con idempotencia, el servidor reconoce que un pago ya se procesó y no repite el cobro, en el caso de nostros evitaria  registrarse dos veces un producto y duplicar stock.
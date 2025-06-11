import fastify from 'fastify';
import userRoutes from './routes/user';
import healthRoutes from './routes/health';
import loadEnv from './utils/env';

loadEnv();

const serverOptions = {
  logger: true,
};

const server = fastify(serverOptions);

server.register(healthRoutes);
server.register(userRoutes);

async function startServer() {
  try {
    const portNumber = Number(process.env.PORT) || 8080;
    const host = process.env.HOST || '127.0.0.1';
    await server.listen({port: portNumber, host});

    const address = server.server.address();
    const port = typeof address === 'string' ? address : address?.port;

    console.log(`Server started at ${host}:${port}`);
  } catch (err) {
    server.log.error('Error starting server', err);
    process.exit(1);
  }
}

startServer();

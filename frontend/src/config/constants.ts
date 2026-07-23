export const API_CONFIG = {
  baseUrl: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
};

export const AUTH_CONFIG = {
  tokenKey: 'auth_token',
  userKey: 'user_data',
};

// ✅ Exporta las variables directamente
export const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

/*docker compose down
   docker compose build backend frontend
   docker compose up -d


   docker compose build backend --no-cache
docker compose up -d backend
docker compose logs -f backend


   Debug YML: docker compose config
   
   */
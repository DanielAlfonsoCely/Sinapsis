# 🏥 Sinapsis

> Unified electronic health record platform for the Colombian healthcare sector, with an AI-powered medical imaging microservice and an integrated RAG-based clinical assistant — Software Engineering II project, Universidad Nacional de Colombia, 2026.

Sinapsis centralizes patient records, appointment scheduling, medical prescriptions, and consultation history in a single environment. It integrates an asynchronous AI microservice (MONAI) for medical image analysis, and a conversational AI assistant powered by Dify + Ollama for real-time clinical support — designed so that if any AI service goes down, the rest of the platform keeps running normally.

---

## ✨ Features

- 🔐 **Role-based access control** — dedicated interfaces for doctors and platform admins (JWT)
- 🧑‍⚕️ **Patient management** — atomic registration of user, profile, and clinical record in one transaction
- 📋 **Clinical history** — consultations, medical prescriptions, referrals, and attachments
- 📅 **Appointment scheduling** — between patients and doctors
- 🤖 **AI medical imaging** — asynchronous analysis via MONAI models (spleen CT segmentation, brain tumor MRI, breast density X-ray classification)
- 💬 **AI clinical assistant** — RAG-based conversational assistant via Dify + Ollama, accessible from the patient interface
- 📝 **Immutable audit log** — every access, successful or not, is recorded via database trigger
- 📄 **PDF export** — clinical history export compliant with Colombian health regulation (Resolución 1995/1999)

---

## 🏗️ Architecture

```
Frontend (Next.js SSR)
        │
        ├──► Backend (Go/Gin) ──► RabbitMQ ──► Python/MONAI Microservice
        │           │
        │           └──► Dify API ──► Ollama + Vector DB (RAG)
        │
        ▼
   PostgreSQL + Audit
```

Four independent subsystems: a Go backend, a Next.js frontend, a Python inference microservice, and a Dify-based RAG assistant. The backend publishes image analysis requests to RabbitMQ; the MONAI worker consumes and responds asynchronously. The Dify assistant is called synchronously from the Go backend via REST and exposed to the patient frontend through a dedicated chat endpoint.

---

## 🛠️ Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.26 + Gin |
| Frontend | Next.js 16 + React 19 + TypeScript |
| Database | PostgreSQL (UUID, JSONB, ENUM, triggers) |
| AI Microservice | Python 3.12 + MONAI + PyTorch |
| AI Assistant | Dify (self-hosted) + Ollama + nomic-embed-text |
| Messaging | RabbitMQ |
| Infrastructure | Docker + Docker Compose |
| Auth | JWT (HS256) + bcrypt |
| UI | Tailwind CSS 4 + Radix UI |

---

## 🤖 AI Models

### Medical Imaging (MONAI)

| Analysis Type | Modality | Task |
|---|---|---|
| `ct_spleen_segmentation` | CT scan | Spleen segmentation |
| `ct_lung_nodule_detection` | CT scan | Lung nodule detection |
| `mri_brain_tumor_segmentation` | MRI | Brain tumor segmentation |
| `xr_breast_density_classification` | X-ray | Breast density classification |

> The AI service provides diagnostic support only — it does not make clinical decisions. Doctors must register a pre-diagnosis before accessing AI suggestions to avoid anchoring bias.

### Clinical Assistant (Dify + RAG)

The patient interface includes a conversational AI assistant built with **Dify** (self-hosted), **Ollama** as the LLM runtime, and **nomic-embed-text** for semantic embeddings. The assistant answers clinical questions using a knowledge base of medical documents indexed via RAG (Retrieval-Augmented Generation).

**Flow:**
```
Patient chat (Next.js) → POST /api/v1/chat (Go/Gin) → Dify REST API → Ollama LLM + Vector DB → Response
```

**Key configuration:**
- LLM: `qwen2.5:14b` via Ollama (local, no external API)
- Embedding model: `nomic-embed-text`
- Retrieval: Hybrid Search (Semantic 0.6 / Keyword 0.4), Top K = 3
- Knowledge base: medical PDFs indexed with chunk size 2000 / overlap 200
- All services run inside Docker; `DIFY_BASE_URL` points to the host IP so the Go container can reach the Dify instance

For Dify + Ollama local setup and configuration, check out: https://github.com/DanielAlfonsoCely/Dify-Chatbot-Documentation

---

## 🚀 Getting Started

### Prerequisites
- Docker Engine 24+ and Docker Compose v2
- Ollama installed and running on the host (`ollama serve`)
- Dify running locally (self-hosted Docker deployment)

### Environment variables

Set the following in `docker-compose.yml` under the backend service:

```yaml
environment:
  # Use host.docker.internal on Mac/Windows with Docker Desktop
  # If that doesn't resolve, replace with your host machine's IP
  DIFY_BASE_URL: http://host.docker.internal/v1
  DIFY_API_KEY: your-dify-api-key-here
```

### Run

```bash
docker compose up --build -d
```

The schema and seed data (`schema.sql`, `init.sql`) load automatically on first run. The first startup of `sinapsis-ai` may be slow while MONAI bundles download — requests queue in RabbitMQ and the rest of the platform is unaffected. The Dify assistant is available immediately once Ollama models are pulled.

---

## 🎨 Design Patterns

Repository, Observer (audit), Middleware chain, Producer-Consumer, Hexagonal Architecture, Strategy (per-model bundle adapters), Dependency Injection, Component-based UI.

---

## 📸 Screenshots

**1. Patient portal — agenda and appointment scheduling**

<img width="466" alt="Patient portal" src="https://github.com/user-attachments/assets/13a8ed87-7dd9-49f8-8ebe-e887a8edfd83" />

**2. Doctor dashboard — daily summary and patient list**

<img width="1462" height="733" alt="Image" src="https://github.com/user-attachments/assets/26bec971-aebf-4cbf-bf4e-6f6f1145aeac" />

**3. Clinical history — unified immutable record**

<img width="1545" alt="Clinical history" src="https://github.com/user-attachments/assets/9592e761-72f1-4fc9-b512-5963873d3c38" />

**4. AI medical imaging analysis (MONAI)**

<img width="1295" alt="MONAI analysis" src="https://github.com/user-attachments/assets/120ded4e-6864-4a4d-9a72-954fbd614392" />

**5. AI clinical assistant — RAG-based chat (Dify + Ollama)**

<img width="1575" alt="AI clinical assistant" src="https://github.com/user-attachments/assets/5334fc23-875e-4fe2-a60d-426491d4c465" />

**6. PDF export — clinical history (Resolución 1995/1999)**

<img width="557" alt="PDF export" src="https://github.com/user-attachments/assets/db09828d-7c0c-46d5-9d6e-a483f2dfca16" />

**7. Admin panel — system audit log**

<img width="1427" alt="Audit log" src="https://github.com/user-attachments/assets/41d3f4d7-7653-4cef-becf-7da940a59bad" />

---

## 👥 Team

Developed as a final project for the **Software Engineering II** course at **Universidad Nacional de Colombia** — Team Error 418.

| Name |
|---|
| Tomás Alejandro Bermúdez Guaqueta |
| Daniel Alfonso Cely Infante |
| Juan Sebastián Gámez Ariza |
| David Alejandro Herrera Novoa |
| Adrian Alberto Diosa Benavides |

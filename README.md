# 🏥 Sinapsis

> Unified electronic health record platform for the Colombian healthcare sector, with an AI-powered medical imaging microservice — Software Engineering II project, Universidad Nacional de Colombia, 2026.

Sinapsis centralizes patient records, appointment scheduling, medical prescriptions, and consultation history in a single environment. It integrates an asynchronous AI microservice (MONAI) for medical image analysis, designed so that if the AI service goes down, the rest of the platform keeps running normally.

---

## ✨ Features

- 🔐 **Role-based access control** — dedicated interfaces for doctors and platform admins (JWT)
- 🧑‍⚕️ **Patient management** — atomic registration of user, profile, and clinical record in one transaction
- 📋 **Clinical history** — consultations, medical prescriptions, referrals, and attachments
- 📅 **Appointment scheduling** — between patients and doctors
- 🤖 **AI medical imaging** — asynchronous analysis via MONAI models (spleen CT segmentation, brain tumor MRI, breast density X-ray classification)
- 📝 **Immutable audit log** — every access, successful or not, is recorded via database trigger
- 📄 **PDF export** — clinical history export compliant with Colombian health regulation (Resolución 1995/1999)

---

## 🏗️ Architecture

```
Frontend (Next.js SSR)
        │
        ▼
   Backend (Go/Gin) ──► RabbitMQ ──► AI Microservice (MONAI/Python)
        │
        ▼
   PostgreSQL + Audit
```

Three independent subsystems: a Go backend, a Next.js frontend, and a Python inference microservice. The backend publishes image analysis requests to RabbitMQ; the MONAI worker consumes and responds asynchronously without blocking the original HTTP request.

---

## 🛠️ Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.26 + Gin |
| Frontend | Next.js 16 + React 19 + TypeScript |
| Database | PostgreSQL (UUID, JSONB, ENUM, triggers) |
| AI Microservice | Python 3.12 + MONAI + PyTorch |
| Messaging | RabbitMQ |
| Infrastructure | Docker + Docker Compose |
| Auth | JWT (HS256) + bcrypt |
| UI | Tailwind CSS 4 + Radix UI |

---

## 🤖 AI Models

| Analysis Type | Modality | Task |
|---|---|---|
| `ct_spleen_segmentation` | CT scan | Spleen segmentation |
| `ct_lung_nodule_detection` | CT scan | Lung nodule detection |
| `mri_brain_tumor_segmentation` | MRI | Brain tumor segmentation |
| `xr_breast_density_classification` | X-ray | Breast density classification |

> The AI service provides diagnostic support only — it does not make clinical decisions. Doctors must register a pre-diagnosis before accessing AI suggestions to avoid anchoring bias.

---

## 🚀 Getting Started

### Prerequisites
- Docker Engine 24+ and Docker Compose v2

### Run

```bash
docker compose up --build -d
```

The schema and seed data (`schema.sql`, `init.sql`) load automatically on first run. The first startup of `sinapsis-ai` may be slow while MONAI bundles download — requests queue in RabbitMQ and the rest of the platform is unaffected.

---

## 🎨 Design Patterns

Repository, Observer (audit), Middleware chain, Producer-Consumer, Hexagonal Architecture, Strategy (per-model bundle adapters), Dependency Injection, Component-based UI.

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

<h1 align="center"> Taskio<img src="https://img.icons8.com/ios-filled/50/3B82F6/task.png" width="28"/></h1>

### 📷 Project Screenshots:
<img width="1898" height="893" alt="s1" src="https://github.com/user-attachments/assets/1e60d139-5216-4521-a2e5-7a1c2e5357da" />
<img width="1901" height="913" alt="s2" src="https://github.com/user-attachments/assets/6e53894c-f8ab-4311-9f7b-c73367ffaa05" />
<img width="1913" height="871" alt="s4" src="https://github.com/user-attachments/assets/98940fb1-767e-4e2e-817d-36c6c9f70c26" />
<img width="1904" height="905" alt="s3" src="https://github.com/user-attachments/assets/d73d9dee-4a97-41f8-bd2e-dc02b402a71f" />


## ⚙️ Features

Here are some of the key features of the Taskio platform:

-  User registration and login (manager and member roles)
-  Project management (create, delete, assign members)
-  Task management (create, delete, update status: pending, in progress, done)
-  Visual workflow representation using task dependency graph
-  Real-time notifications for project and task events
-  Document upload and activity history tracking
-  Advanced analytics and task status tracking over time
-  Input validation, security mechanisms, CQRS + Event Sourcing
-  Containerization with Docker and orchestration via Kubernetes
-  CI/CD pipeline for automated deployment
-  And more


## 💻 Built with

Technologies and tools used in the development of Taskio:

- ⚙️ Go (Golang) – for building backend microservices
- 🗃️ MongoDB, Cassandra, Neo4j – for document, wide-column, and graph data storage
- 📦 Redis, HDFS – for caching and document storage
- 🌐 REST APIs – exposed through an API Gateway
- 🐳 Docker & Docker Compose – for containerization
- 🛡️ HTTPS, RBAC, authentication & input validation – for security
- 🔄 CQRS, Saga pattern, Event Sourcing – for handling business logic and state changes
- 💡 Frontend Angular

## 🔧 Installation Steps:

1. Cloning a repository
```bash
git clone https://github.com/veljkoo10/taskboard-taskio.git
```
2. Build and run with Docker Compose
Make sure you have Docker and Docker Compose installed.
```bash
docker-compose up --build
```
3. Access the application
After all containers are up and running, open your browser:
```bash
http://localhost:8080
```
📕 Prerequisites
Make sure you have the following tools installed before running the project:
<ul>
  <li>Docker</li>
  <li>Docker Compose</li>
  <li>Go</li>
  <li>Git</li>
  <li>Any code editor or IDE (e.g., VS Code, GoLand)</li>
</ul>


🚀 Run the Application:
If you prefer to run microservices individually without Docker:

1.Navigate to the desired service directory:
```bash
cd services/users
go run main.go
```
2.Repeat for each microservice (users, tasks, projects, etc.).

3.Make sure your local databases are running and configs are set via .env files.


Once the application window opens, you can interact with it using the provided user interface.
## 👩‍👨‍👦‍👧 Contributors:
- [veljkoo10](https://github.com/veljkoo10)
- [SekulaProgramer2023](https://github.com/SekulaProgramer2023)
- [rastkokupusovic](https://github.com/rastkokupusovic)
- [StojkovSR11](https://github.com/StojkovSR11)

I want to build a full-stack To-Do List application for a ShopBack AI-assisted engineering take-home assignment.

My goal is to demonstrate practical software engineering skills, including requirement analysis, architecture design, implementation, testing, debugging, and responsible AI-assisted development.

Tech stack:

- Frontend: React, TypeScript, Vite
- Backend: Go, Gin
- Database: PostgreSQL
- Database driver: pgx
- Authentication: JWT with HttpOnly cookies
- Database migrations: golang-migrate
- Local development: Docker Compose

Required features:

1. User registration
2. User login and logout
3. Authentication-protected task management
4. Create, read, update, and delete tasks
5. Mark tasks as completed or active
6. Filter tasks by All, Active, and Completed
7. Persist users and tasks in PostgreSQL
8. Ensure users can only access and modify their own tasks

Engineering requirements:

- Use a modular monolith architecture for the backend
- Separate handlers, services, repositories, models, and middleware
- Use proper TypeScript types on the frontend
- Validate user input on the backend
- Hash passwords securely
- Handle errors consistently
- Use environment variables for configuration
- Include database migrations
- Keep the implementation appropriate for a small take-home assignment
- Avoid unnecessary dependencies and overengineering

Before implementing:

1. Analyze the requirements
2. Propose a concise architecture and project structure
3. Identify important security concerns and edge cases
4. Break the implementation into small, reviewable tasks

Implement one task at a time. Explain important design decisions, show which files are changed, and wait for my review before proceeding to the next task.

Do not implement the entire application in one response. I want to understand, review, test, and take ownership of every part of the code.
FROM node:16-alpine
WORKDIR /app

RUN addgroup -S mercari && adduser -S trainee -G mercari
USER trainee

# Copy package.json and package-lock.json to the Docker image
COPY package.json package-lock.json ./

# Install dependencies
RUN npm install

# Copy the entire project directory to the Docker image
COPY . .

# Build the React app
RUN npm run build


CMD ["npm", "run", "serve"]

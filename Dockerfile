# Use an official Node runtime as a base image
FROM node:22

# Set the working directory in the container
WORKDIR /usr/src/app

# Copy package.json and package-lock.json to the working directory
COPY package*.json ./

# Install Angular CLI globally
#RUN npm install -g @angular/cli

# Install dependencies
RUN npm install

# Copy the remaining application code to the working directory
COPY . .

# Change ownership of the app directory to the node user
RUN chown -R node:node /usr/src/app

# Switch to non-root user
USER node

# EXPOSE 4200

# EXPOSE 80

# Install NGINX for reverse proxying
# RUN apt-get update && apt-get install -y nginx

# Remove the default NGINX configuration
# RUN rm /etc/nginx/nginx.conf

# Start Angular development server using ng serve
CMD ["sh", "-c", "npm run start"]
# CMD ["sh", "-c", "nginx -c /etc/nginx/nginx.conf -g 'daemon off;' & npm run start"]


# ********************************* Docker Commands *********************************


# command to build docker image to be run in the root folder
# docker build -t ssd-ui .

# command to create and run a docker container
# docker run -it --network=host -v /home/satya/Development/oes-ui-oct-current/oes-ui/:/usr/src/app ssd-ui

# command to fetch the stopped container id
# docker ps -a

# command to stop the docker container
# docker stop <container_id>

# command to start the stopped docker container
# docker start <container_id>

# command to check the docker logs
# docker logs --tail 1000 -f <container_id>
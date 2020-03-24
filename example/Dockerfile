
FROM golang:1.13

RUN wget -q -O - https://dl-ssl.google.com/linux/linux_signing_key.pub | apt-key add -
RUN sh -c 'echo "deb [arch=amd64] http://dl.google.com/linux/chrome/deb/ stable main" >> /etc/apt/sources.list.d/google-chrome.list'
RUN apt-get update
RUN apt-get -y install xvfb ffmpeg unzip google-chrome-stable --no-install-recommends --allow-unauthenticated

RUN wget https://chromedriver.storage.googleapis.com/80.0.3987.106/chromedriver_linux64.zip && unzip chromedriver_linux64.zip
RUN mv chromedriver /usr/bin/chromedriver
RUN chown root:root /usr/bin/chromedriver
RUN chmod +x /usr/bin/chromedriver
RUN apt-get autoremove && apt-get clean && rm chromedriver_linux64.zip

ADD . ytuploader
RUN cd ytuploader && go mod tidy
RUN ls
WORKDIR ytuploader/example
CMD [ "go", "run", "ytuploader.go"]
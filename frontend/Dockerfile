# Copyright 2023 Google LLC

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     https://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM node:20.5.0-bullseye as base

RUN groupadd -r appuser && useradd -r -g appuser -G audio,video appuser \
    && mkdir -p /home/appuser/Downloads \
    && chown -R appuser:appuser /home/appuser

FROM base as base-test

# Cache all of the dependencies early to prevent re-download
RUN apt-get update \
    && apt-get install -y \
        # Install Chrome dependencies
        # https://github.com/GoogleChrome/developer.chrome.com/blob/main/site/en/docs/puppeteer/troubleshooting/index.md
        ca-certificates \
        fonts-liberation \
        libappindicator3-1 \
        libasound2 \
        libatk-bridge2.0-0 \
        libatk1.0-0 \
        libc6 \
        libcairo2 \
        libcups2 \
        libdbus-1-3 \
        libexpat1 \
        libfontconfig1 \
        libgbm1 \
        libgcc1 \
        libglib2.0-0 \
        libgtk-3-0 \
        libnspr4 \
        libnss3 \
        libpango-1.0-0 \
        libpangocairo-1.0-0 \
        libstdc++6 \
        libx11-6 \
        libx11-xcb1 \
        libxcb1 \
        libxcomposite1 \
        libxcursor1 \
        libxdamage1 \
        libxext6 \
        libxfixes3 \
        libxi6 \
        libxrandr2 \
        libxrender1 \
        libxss1 \
        libxtst6 \
        lsb-release \
        wget \
        xdg-utils \
        # Puppeteer dependencies
        # https://github.com/puppeteer/puppeteer/blob/main/docs/troubleshooting.md#running-puppeteer-in-docker
        fonts-ipafont-gothic \
        fonts-wqy-zenhei \
        fonts-thai-tlwg \
        fonts-kacst \
        fonts-freefont-ttf
# https://github.com/puppeteer/puppeteer/blob/main/docs/troubleshooting.md#recommended-enable-user-namespace-cloning
RUN set -ex \
    && CHROME_LOCATION=`npx @puppeteer/browsers install chrome@stable | awk '{print $2}'` \
    && ln -s $CHROME_LOCATION /usr/bin/chrome \
    && chrome --version \
    # Format is 'Google Chrome for Testing a.b.c.d'
    && CHROME_VERSION=`chrome --version | awk -F '[ .]' '{print $5"."$6"."$7"."$8}'` \
    && CHROMEDRIVER_LOCATION=`npx @puppeteer/browsers install chromedriver@$CHROME_VERSION | awk '{print $2}'` \
    && ln -s $CHROMEDRIVER_LOCATION /usr/bin/chromedriver \
    && chromedriver --version

ENV CHROME_PATH=/usr/bin/chrome


FROM base as app

USER appuser
WORKDIR /app
COPY --chown=appuser:appuser package.json package-lock.json /app/
RUN npm install

COPY --chown=appuser:appuser . /app/

EXPOSE 8000
CMD npm run start

FROM base-test as test

USER appuser
COPY --from=app --chown=appuser:appuser /app /app
WORKDIR /app

CMD npm run test

# A minimal Git server for the throwaway cluster: CI pushes the commit under
# test here and Argo CD syncs from it, so no external Git credentials are needed.
FROM alpine:3.22
RUN apk add --no-cache git git-daemon \
 && adduser -D -u 10001 git
USER 10001
EXPOSE 9418
ENTRYPOINT ["sh", "-c", "git init -q --bare -b main /srv/git/gantry.git && git -C /srv/git/gantry.git config daemon.receivepack true && exec git daemon --reuseaddr --base-path=/srv/git --export-all --enable=receive-pack /srv/git"]

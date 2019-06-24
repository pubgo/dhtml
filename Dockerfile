FROM rendora/chrome-headless
COPY main /dhtml
ENTRYPOINT ["/dhtml"]
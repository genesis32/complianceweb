FROM alpine
RUN mkdir -p /templates/html/ /static/images /static/js /static/css
COPY templates/html/* /templates/html/
COPY static/css/* /static/css/
COPY static/js/* /static/js/
COPY static/images/* /static/images/
ADD enterpriseportal2 /
ENV PORT 8080
ENV ENV prod
ENV ENTERPRISEPORTAL2_POSTGRES_USER ep2
ENV ENTERPRISEPORTAL2_POSTGRES_PASSWORD ep2
ENV ENTERPRISEPORTAL2_POSTGRES_DBNAME enterpriseportal2
CMD ["/enterpriseportal2"]
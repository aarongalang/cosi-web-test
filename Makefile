deploy: push
	kubectl apply -f web-deployment.yaml
push:
	docker build . -t aarongalang/cosi-web-test:latest
	docker push aarongalang/cosi-web-test:latest
port:
	kubectl port-forward deployment/cosi-web-deployment 8080:8080
clean:
	docker image rm aarongalang/cosi-web-test
	kubectl delete -f web-deployment.yaml
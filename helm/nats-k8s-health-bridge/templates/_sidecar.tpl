{{- define "nkhb.sidecarExample" -}}
{{- if and .Values.nats.credentialsSecret .Values.nats.nkeySecret -}}
{{- fail "nats.credentialsSecret and nats.nkeySecret are mutually exclusive" -}}
{{- end -}}
{{- if and .Values.nats.tlsClientAuth (not .Values.nats.tlsSecret) -}}
{{- fail "nats.tlsClientAuth=true requires nats.tlsSecret" -}}
{{- end -}}
{{- range $name := list .Values.nats.credentialsSecret .Values.nats.nkeySecret .Values.nats.tlsSecret .Values.server.metricsTokenSecret .Values.server.statusTokenSecret -}}
{{- if and $name (not (regexMatch "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$" $name)) -}}
{{- fail (printf "invalid Kubernetes Secret name: %s" $name) -}}
{{- end -}}
{{- end }}
# COPY the containers[] entry and volumes[] entries into your existing Pod template.
# This chart creates only a ConfigMap, not a workload or Secret.
containers:
  - name: nats-health-bridge
    image: {{ printf "%s:%s" .Values.image.repository .Values.image.tag | quote }}
    imagePullPolicy: {{ .Values.image.pullPolicy }}
    args: ["--config", "/etc/nats-health-bridge/config.yaml"]
    ports:
      - containerPort: {{ .Values.port }}
        name: health
{{- if or .Values.server.metricsTokenSecret .Values.server.statusTokenSecret .Values.nats.credentialsSecret .Values.nats.nkeySecret .Values.nats.tlsSecret }}
    env:
{{- if .Values.server.metricsTokenSecret }}
      - name: NKHB_SERVER_METRICS_TOKEN
        valueFrom:
          secretKeyRef:
            name: {{ .Values.server.metricsTokenSecret | quote }}
            key: token
{{- end }}
{{- if .Values.server.statusTokenSecret }}
      - name: NKHB_SERVER_STATUS_TOKEN
        valueFrom:
          secretKeyRef:
            name: {{ .Values.server.statusTokenSecret | quote }}
            key: token
{{- end }}
{{- if .Values.nats.credentialsSecret }}
      - name: NKHB_NATS_CREDENTIALS_FILE
        value: /etc/nats-health-bridge-creds/user.creds
{{- end }}
{{- if .Values.nats.nkeySecret }}
      - name: NKHB_NATS_NKEY_FILE
        value: /etc/nats-health-bridge-nkey/user.nkey
{{- end }}
{{- if .Values.nats.tlsSecret }}
      - name: NKHB_NATS_TLS_CA
        value: /etc/nats-health-bridge-tls/ca.crt
{{- if .Values.nats.tlsClientAuth }}
      - name: NKHB_NATS_TLS_CERT
        value: /etc/nats-health-bridge-tls/tls.crt
      - name: NKHB_NATS_TLS_KEY
        value: /etc/nats-health-bridge-tls/tls.key
{{- end }}
{{- end }}
{{- end }}
    securityContext:
      runAsNonRoot: true
      runAsUser: 65532
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      capabilities:
        drop: ["ALL"]
      seccompProfile:
        type: RuntimeDefault
    volumeMounts:
      - name: nats-health-bridge-config
        mountPath: /etc/nats-health-bridge
        readOnly: true
{{- if .Values.nats.credentialsSecret }}
      - name: nats-health-bridge-creds
        mountPath: /etc/nats-health-bridge-creds
        readOnly: true
{{- end }}
{{- if .Values.nats.nkeySecret }}
      - name: nats-health-bridge-nkey
        mountPath: /etc/nats-health-bridge-nkey
        readOnly: true
{{- end }}
{{- if .Values.nats.tlsSecret }}
      - name: nats-health-bridge-tls
        mountPath: /etc/nats-health-bridge-tls
        readOnly: true
{{- end }}
    livenessProbe:
      httpGet:
        path: /healthz
        port: health
      initialDelaySeconds: {{ .Values.probes.liveness.initialDelaySeconds }}
      periodSeconds: {{ .Values.probes.liveness.periodSeconds }}
      failureThreshold: {{ .Values.probes.liveness.failureThreshold }}
    readinessProbe:
      httpGet:
        path: /readyz
        port: health
      initialDelaySeconds: {{ .Values.probes.readiness.initialDelaySeconds }}
      periodSeconds: {{ .Values.probes.readiness.periodSeconds }}
      failureThreshold: {{ .Values.probes.readiness.failureThreshold }}
    resources:
{{ toYaml .Values.resources | indent 6 }}
volumes:
  - name: nats-health-bridge-config
    configMap:
      name: {{ include "nats-k8s-health-bridge.fullname" . | quote }}
{{- if .Values.nats.credentialsSecret }}
  - name: nats-health-bridge-creds
    secret:
      secretName: {{ .Values.nats.credentialsSecret | quote }}
      items:
        - key: user.creds
          path: user.creds
{{- end }}
{{- if .Values.nats.nkeySecret }}
  - name: nats-health-bridge-nkey
    secret:
      secretName: {{ .Values.nats.nkeySecret | quote }}
      items:
        - key: user.nkey
          path: user.nkey
{{- end }}
{{- if .Values.nats.tlsSecret }}
  - name: nats-health-bridge-tls
    secret:
      secretName: {{ .Values.nats.tlsSecret | quote }}
      items:
        - key: ca.crt
          path: ca.crt
{{- if .Values.nats.tlsClientAuth }}
        - key: tls.crt
          path: tls.crt
        - key: tls.key
          path: tls.key
{{- end }}
{{- end }}
{{- end -}}

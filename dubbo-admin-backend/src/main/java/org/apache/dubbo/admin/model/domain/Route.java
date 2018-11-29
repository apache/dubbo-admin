package org.apache.dubbo.admin.model.domain;

public abstract class Route {

    private String key;
    private boolean runtime;
    private boolean force;
    private boolean enabled;
    private int priority;

    public String getKey() {
        return key;
    }

    public void setKey(String key) {
        this.key = key;
    }

    public boolean isRuntime() {
        return runtime;
    }

    public void setRuntime(boolean runtime) {
        this.runtime = runtime;
    }

    public boolean isForce() {
        return force;
    }

    public void setForce(boolean force) {
        this.force = force;
    }

    public boolean isEnabled() {
        return enabled;
    }

    public void setEnabled(boolean enabled) {
        this.enabled = enabled;
    }

    public int getPriority() {
        return priority;
    }

    public void setPriority(int priority) {
        this.priority = priority;
    }
}

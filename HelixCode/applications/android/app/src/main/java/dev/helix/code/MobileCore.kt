package dev.helix.code

import org.json.JSONObject

object MobileCore {
    val shared = MobileCoreInstance()
}

class MobileCoreInstance {
    private var core: HelixCoreMobileCore? = null

    init {
        core = HelixCoreNewMobileCore()
    }

    fun initialize() {
        core?.initialize()
    }

    fun connect(serverURL: String, username: String, password: String): Boolean {
        return try {
            core?.connect(serverURL, username, password)
            true
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }

    fun disconnect() {
        try {
            core?.disconnect()
        } catch (e: Exception) {
            e.printStackTrace()
        }
    }

    val isConnected: Boolean
        get() = core?.isConnected() ?: false

    val currentUser: String
        get() = core?.currentUser ?: ""

    val dashboardData: String
        get() = core?.dashboardData ?: "{}"

    val tasks: String
        get() = core?.tasks ?: "[]"

    val workers: String
        get() = core?.workers ?: "[]"

    fun createTask(name: String, description: String): String {
        return core?.createTask(name, description) ?: "{\"error\": \"Core not initialized\"}"
    }

    fun sendNotification(title: String, message: String, type: String): Boolean {
        return try {
            val result = core?.sendNotification(title, message, type)
            val json = JSONObject(result ?: "{}")
            json.optBoolean("success", false)
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }

    val theme: String
        get() = core?.theme ?: "{}"

    fun setTheme(themeName: String): Boolean {
        return core?.setTheme(themeName) ?: false
    }

    val availableThemes: String
        get() = core?.availableThemes ?: "[]"

    fun close() {
        try {
            core?.close()
        } catch (e: Exception) {
            e.printStackTrace()
        }
    }
}
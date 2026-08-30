package uts.sdk.modules.akPush

import android.Manifest
import android.app.Activity
import android.app.NotificationManager
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import android.os.Handler
import android.os.Looper
import android.provider.Settings
import io.dcloud.uts.UTSAndroid
import org.json.JSONObject
import java.lang.reflect.Proxy
import java.util.Locale

/**
 * Reflection keeps the Google and China source sets mutually compilable. The
 * custom-base generator is responsible for including exactly one dependency
 * set; absence of a class means that provider is unavailable, never fallback.
 */
object AkPushAndroidBridge {
    private const val META_VARIANT = "com.appkernia.push.variant"
    private val main = Handler(Looper.getMainLooper())

    @JvmStatic fun buildVariant(): String = metadata(META_VARIANT).ifBlank { "unconfigured" }

    @JvmStatic fun provider(): String {
        val variant = buildVariant()
        if (variant == "google") return if (available("com.google.firebase.messaging.FirebaseMessaging")) "fcm" else ""
        if (variant != "china") return ""
        val maker = Build.MANUFACTURER.lowercase(Locale.ROOT)
        return when {
            maker.contains("honor") && available("com.hihonor.push.sdk.HonorPushClient") -> "honor"
            maker.contains("huawei") && available("com.huawei.hms.aaid.HmsInstanceId") -> "huawei_android"
            maker.contains("xiaomi") && available("com.xiaomi.mipush.sdk.MiPushClient") -> "xiaomi"
            (maker.contains("oppo") || maker.contains("oneplus") || maker.contains("realme")) && available("com.heytap.msp.push.HeytapPushManager") -> "oppo"
            (maker.contains("vivo") || maker.contains("iqoo")) && available("com.vivo.push.PushClient") -> "vivo"
            maker.contains("meizu") && available("com.meizu.cloud.pushsdk.PushManager") -> "meizu"
            else -> ""
        }
    }

    @JvmStatic fun authorizationStatus(): String {
        val context = context() ?: return "unavailable"
        val manager = context.getSystemService(Context.NOTIFICATION_SERVICE) as? NotificationManager ?: return "unavailable"
        if (!manager.areNotificationsEnabled()) return "denied"
        if (Build.VERSION.SDK_INT >= 33 && context.checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED) return "not_determined"
        return "authorized"
    }

    @JvmStatic fun requestAuthorization(callback: (Boolean) -> Unit) {
        val activity = UTSAndroid.getUniActivity() as? Activity ?: run { callback(false); return }
        if (Build.VERSION.SDK_INT < 33 || activity.checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED) { callback(true); return }
        activity.requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), 0x4A50)
        pollAuthorization(0, callback)
    }

    @JvmStatic fun register(success: (String, String) -> Unit, failure: (String) -> Unit) {
        if (authorizationStatus() != "authorized") { failure("permission_denied"); return }
        when (provider()) {
            "fcm" -> firebase(success, failure)
            "huawei_android" -> huawei(success, failure)
            "honor" -> honor(success, failure)
            "xiaomi" -> xiaomi(success, failure)
            "oppo" -> oppo(success, failure)
            "vivo" -> vivo(success, failure)
            "meizu" -> meizu(success, failure)
            else -> failure("adapter_unavailable")
        }
    }

    @JvmStatic fun unregister(): Boolean {
        // Server binding is disabled first. Native deletion is best effort and
        // provider-specific; a future token callback remains safe and rebinds
        // only after a fresh explicit opt-in.
        return provider().isNotBlank()
    }

    @JvmStatic fun takeNotificationOpen(): String {
        val activity = UTSAndroid.getUniActivity() as? Activity ?: return ""
        val intent = activity.intent ?: return ""
        if (intent.getBooleanExtra("com.appkernia.push.consumed", false)) return ""
        val values = mutableMapOf<String, String>()
        val extras = intent.extras
        extras?.keySet()?.forEach { key ->
            val raw = extras.get(key)
            if (raw is String) values[key] = raw
        }
        val candidates = listOf("data", "payload", "mipush_payload", "extraData", "extra", "action_parameters", "parameters")
        for (key in candidates) {
            val raw = values[key].orEmpty()
            if (raw.isBlank()) continue
            runCatching {
                val nested = JSONObject(raw)
                nested.keys().forEach { nestedKey -> if (!values.containsKey(nestedKey)) values[nestedKey] = nested.optString(nestedKey) }
            }
        }
        if (values["schema_version"] != "1" || values["delivery_id"].isNullOrBlank() || values["message_id"].isNullOrBlank()) return ""
        intent.putExtra("com.appkernia.push.consumed", true)
        return JSONObject(values as Map<*, *>).toString()
    }

    private fun firebase(success: (String, String) -> Unit, failure: (String) -> Unit) = guarded(failure) {
        val messaging = Class.forName("com.google.firebase.messaging.FirebaseMessaging").getMethod("getInstance").invoke(null)
        val task = messaging.javaClass.getMethod("getToken").invoke(messaging)
        val listenerClass = Class.forName("com.google.android.gms.tasks.OnCompleteListener")
        val listener = Proxy.newProxyInstance(listenerClass.classLoader, arrayOf(listenerClass)) { _, method, args ->
            if (method.name == "onComplete") {
                val completed = args?.get(0) ?: return@newProxyInstance null
                val ok = completed.javaClass.getMethod("isSuccessful").invoke(completed) as Boolean
                val token = if (ok) completed.javaClass.getMethod("getResult").invoke(completed) as? String else null
                if (!token.isNullOrBlank()) success(token, packageVersion("com.google.firebase.messaging")) else failure("token_failed")
            }
            null
        }
        task.javaClass.getMethod("addOnCompleteListener", listenerClass).invoke(task, listener)
    }

    private fun huawei(success: (String, String) -> Unit, failure: (String) -> Unit) = guarded(failure) {
        val context = context()!!
        val instanceClass = Class.forName("com.huawei.hms.aaid.HmsInstanceId")
        val instance = instanceClass.getMethod("getInstance", Context::class.java).invoke(null, context)
        val appId = metadata("com.appkernia.push.huawei.app_id")
        if (appId.isBlank()) { failure("configuration_missing"); return@guarded }
        val token = instanceClass.getMethod("getToken", String::class.java, String::class.java).invoke(instance, appId, "HCM") as? String
        if (!token.isNullOrBlank()) success(token, packageVersion("com.huawei.hms.push")) else failure("token_failed")
    }

    private fun honor(success: (String, String) -> Unit, failure: (String) -> Unit) = guarded(failure) {
        val clientClass = Class.forName("com.hihonor.push.sdk.HonorPushClient")
        val client = clientClass.getMethod("getInstance").invoke(null)
        val callbackClass = Class.forName("com.hihonor.push.sdk.HonorPushCallback")
        val callback = callback(callbackClass, success, failure, "com.hihonor.push")
        clientClass.getMethod("getPushToken", callbackClass).invoke(client, callback)
    }

    private fun xiaomi(success: (String, String) -> Unit, failure: (String) -> Unit) = guarded(failure) {
        val context = context()!!
        val client = Class.forName("com.xiaomi.mipush.sdk.MiPushClient")
        val appId = metadata("com.appkernia.push.xiaomi.app_id")
        val appKey = metadata("com.appkernia.push.xiaomi.app_key")
        if (appId.isBlank() || appKey.isBlank()) { failure("configuration_missing"); return@guarded }
        client.getMethod("registerPush", Context::class.java, String::class.java, String::class.java).invoke(null, context, appId, appKey)
        pollToken(0, { client.getMethod("getRegId", Context::class.java).invoke(null, context) as? String }, { success(it, packageVersion("com.xiaomi.xmsf")) }, failure)
    }

    private fun oppo(success: (String, String) -> Unit, failure: (String) -> Unit) = guarded(failure) {
        val context = context()!!
        val manager = Class.forName("com.heytap.msp.push.HeytapPushManager")
        val appKey = metadata("com.appkernia.push.oppo.app_key")
        val appSecret = metadata("com.appkernia.push.oppo.app_secret")
        if (appKey.isBlank() || appSecret.isBlank()) { failure("configuration_missing"); return@guarded }
        manager.getMethod("init", Context::class.java, Boolean::class.javaPrimitiveType).invoke(null, context, false)
        val callbackClass = Class.forName("com.heytap.msp.push.callback.ICallBackResultService")
        val callback = Proxy.newProxyInstance(callbackClass.classLoader, arrayOf(callbackClass)) { _, method, args ->
            if (method.name == "onRegister") {
                val code = args?.get(0) as? Int ?: -1; val token = args?.get(1) as? String
                if (code == 0 && !token.isNullOrBlank()) success(token, packageVersion("com.heytap.mcs")) else failure("token_failed")
            }
            null
        }
        manager.getMethod("register", Context::class.java, String::class.java, String::class.java, callbackClass).invoke(null, context, appKey, appSecret, callback)
    }

    private fun vivo(success: (String, String) -> Unit, failure: (String) -> Unit) = guarded(failure) {
        val context = context()!!
        val clientClass = Class.forName("com.vivo.push.PushClient")
        val client = clientClass.getMethod("getInstance", Context::class.java).invoke(null, context)
        clientClass.getMethod("initialize").invoke(client)
        val listenerClass = Class.forName("com.vivo.push.IPushActionListener")
        val listener = Proxy.newProxyInstance(listenerClass.classLoader, arrayOf(listenerClass)) { _, method, args ->
            if (method.name == "onStateChanged") {
                val code = args?.get(0) as? Int ?: -1
                val token = clientClass.getMethod("getRegId").invoke(client) as? String
                if (code == 0 && !token.isNullOrBlank()) success(token, packageVersion("com.vivo.push")) else failure("token_failed")
            }
            null
        }
        clientClass.getMethod("turnOnPush", listenerClass).invoke(client, listener)
    }

    private fun meizu(success: (String, String) -> Unit, failure: (String) -> Unit) = guarded(failure) {
        val context = context()!!
        val manager = Class.forName("com.meizu.cloud.pushsdk.PushManager")
        val appId = metadata("com.appkernia.push.meizu.app_id")
        val appKey = metadata("com.appkernia.push.meizu.app_key")
        if (appId.isBlank() || appKey.isBlank()) { failure("configuration_missing"); return@guarded }
        manager.getMethod("register", Context::class.java, String::class.java, String::class.java).invoke(null, context, appId, appKey)
        pollToken(0, { manager.getMethod("getPushId", Context::class.java).invoke(null, context) as? String }, { success(it, packageVersion("com.meizu.cloud.pushsdk")) }, failure)
    }

    private fun callback(type: Class<*>, success: (String, String) -> Unit, failure: (String) -> Unit, packageName: String): Any =
        Proxy.newProxyInstance(type.classLoader, arrayOf(type)) { _, method, args ->
            when (method.name) {
                "onSuccess" -> { val token = args?.get(0) as? String; if (!token.isNullOrBlank()) success(token, packageVersion(packageName)) else failure("token_failed") }
                "onFailure" -> failure("token_failed")
            }
            null
        }

    private fun pollAuthorization(attempt: Int, callback: (Boolean) -> Unit) {
        val status = authorizationStatus()
        if (status == "authorized") { callback(true); return }
        if (attempt >= 60) { callback(false); return }
        main.postDelayed({ pollAuthorization(attempt + 1, callback) }, 500)
    }

    private fun pollToken(attempt: Int, read: () -> String?, success: (String) -> Unit, failure: (String) -> Unit) {
        val token = runCatching(read).getOrNull()
        if (!token.isNullOrBlank()) { success(token); return }
        if (attempt >= 30) { failure("token_failed"); return }
        main.postDelayed({ pollToken(attempt + 1, read, success, failure) }, 500)
    }

    private fun metadata(key: String): String {
        val context = context() ?: return ""
        val info = context.packageManager.getApplicationInfo(context.packageName, PackageManager.GET_META_DATA)
        return info.metaData?.getString(key)?.trim().orEmpty()
    }

    private fun context(): Context? = UTSAndroid.getAppContext() as? Context
    private fun available(name: String): Boolean = runCatching { Class.forName(name) }.isSuccess
    private fun packageVersion(name: String): String = runCatching {
        val context = context()!!
        context.packageManager.getPackageInfo(name, 0).versionName ?: "unknown"
    }.getOrDefault("unknown")
    private inline fun guarded(failure: (String) -> Unit, block: () -> Unit) { try { block() } catch (_: Throwable) { failure("native_failure") } }
}

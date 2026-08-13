package com.xbs.app.data.api

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.int
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.Interceptor
import okhttp3.Response

/** 非 2xx + JSON 错误包 → 提取 code/message 抛 ApiException，让 Repository 的 runCatching 捕获。 */
class ErrorEnvelopeInterceptor : Interceptor {
    private val json = Json { ignoreUnknownKeys = true }

    override fun intercept(chain: Interceptor.Chain): Response {
        val response = chain.proceed(chain.request())
        if (response.isSuccessful) return response
        val text = response.body?.string()
        response.body?.close()
        val (code, message) = if (text.isNullOrBlank()) {
            response.code to response.message
        } else {
            runCatching {
                val obj = json.parseToJsonElement(text).jsonObject
                (obj["code"]?.jsonPrimitive?.int ?: response.code) to
                    (obj["message"]?.jsonPrimitive?.content ?: response.message)
            }.getOrElse { response.code to response.message }
        }
        throw ApiException(code, message)
    }
}

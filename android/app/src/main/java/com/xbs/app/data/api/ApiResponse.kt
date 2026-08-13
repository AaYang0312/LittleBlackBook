package com.xbs.app.data.api

import java.io.IOException
import kotlinx.serialization.Serializable

@Serializable
data class ApiResponse<T>(
    val code: Int,
    val message: String,
    val data: T? = null,
)

/** 继承 IOException：OkHttp 异步路径对非 IOException 会包一层 "canceled due to ..."，导致 message 丢失。 */
class ApiException(val code: Int, override val message: String) : IOException(message)

/** code!=0 抛 ApiException；data 缺失抛 ApiException(-1)。用于有 data 的接口。 */
fun <T> ApiResponse<T>.dataOrThrow(): T {
    if (code != 0) throw ApiException(code, message)
    return data ?: throw ApiException(-1, "empty data")
}

/** 用于无 data 的写接口（like/collect/follow/delete）。 */
fun ApiResponse<Unit>.successOrThrow() {
    if (code != 0) throw ApiException(code, message)
}

package com.xbs.app.data.api

import kotlinx.serialization.Serializable

@Serializable
data class ApiResponse<T>(
    val code: Int,
    val message: String,
    val data: T? = null,
)

class ApiException(val code: Int, override val message: String) : Exception(message)

/** code!=0 抛 ApiException；data 缺失抛 ApiException(-1)。用于有 data 的接口。 */
fun <T> ApiResponse<T>.dataOrThrow(): T {
    if (code != 0) throw ApiException(code, message)
    return data ?: throw ApiException(-1, "empty data")
}

/** 用于无 data 的写接口（like/collect/follow/delete）。 */
fun ApiResponse<Unit>.successOrThrow() {
    if (code != 0) throw ApiException(code, message)
}

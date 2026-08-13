package com.xbs.app.data.repository

import com.xbs.app.data.api.NoteApi
import com.xbs.app.data.api.dataOrThrow
import com.xbs.app.data.api.dto.NoteDto
import com.xbs.app.data.api.dto.NotePage
import com.xbs.app.data.api.dto.PublishReq
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.MultipartBody
import okhttp3.RequestBody.Companion.toRequestBody
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class NoteRepository @Inject constructor(
    private val api: NoteApi,
) {
    suspend fun latest(cursor: Long?, size: Int = 20): Result<NotePage> = runCatching {
        api.latest(cursor, size).dataOrThrow()
    }

    suspend fun detail(id: Long): Result<NoteDto> = runCatching { api.detail(id).dataOrThrow() }

    suspend fun publish(title: String, content: String, imageUrls: List<String>): Result<NoteDto> = runCatching {
        api.publish(PublishReq(title, content, imageUrls)).dataOrThrow()
    }

    /** 上传单张图片，返回可访问 URL。 */
    suspend fun uploadImage(bytes: ByteArray, filename: String): Result<String> = runCatching {
        val body = bytes.toRequestBody("image/jpeg".toMediaType())
        val part = MultipartBody.Part.createFormData("file", filename, body)
        api.uploadImage(part).dataOrThrow().url
    }
}

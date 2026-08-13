package com.xbs.app.data.api

import com.xbs.app.data.api.dto.NoteDto
import com.xbs.app.data.api.dto.NotePage
import com.xbs.app.data.api.dto.PublishReq
import com.xbs.app.data.api.dto.UploadResp
import okhttp3.MultipartBody
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.Multipart
import retrofit2.http.POST
import retrofit2.http.Part
import retrofit2.http.Path
import retrofit2.http.Query

interface NoteApi {
    @GET("notes/latest")
    suspend fun latest(@Query("cursor") cursor: Long? = null, @Query("size") size: Int = 20): ApiResponse<NotePage>

    @GET("notes/{id}")
    suspend fun detail(@Path("id") id: Long): ApiResponse<NoteDto>

    @POST("notes")
    suspend fun publish(@Body req: PublishReq): ApiResponse<NoteDto>

    @DELETE("notes/{id}")
    suspend fun delete(@Path("id") id: Long): ApiResponse<Unit>

    @Multipart
    @POST("notes/images")
    suspend fun uploadImage(@Part file: MultipartBody.Part): ApiResponse<UploadResp>
}

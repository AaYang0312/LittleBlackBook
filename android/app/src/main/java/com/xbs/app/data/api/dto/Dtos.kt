package com.xbs.app.data.api.dto

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class UserDto(
    val id: Long,
    val username: String,
    val nickname: String = "",
    @SerialName("avatar_url") val avatarUrl: String = "",
    val bio: String = "",
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class NoteDto(
    val id: Long,
    @SerialName("user_id") val userId: Long,
    val title: String,
    val content: String = "",
    @SerialName("cover_url") val coverUrl: String = "",
    val images: List<String> = emptyList(),
    @SerialName("like_count") val likeCount: Long = 0,
    @SerialName("collect_count") val collectCount: Long = 0,
    @SerialName("comment_count") val commentCount: Long = 0,
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class NotePage(
    val list: List<NoteDto> = emptyList(),
    @SerialName("next_cursor") val nextCursor: Long = 0,
    @SerialName("has_more") val hasMore: Boolean = false,
)

@Serializable
data class CommentDto(
    val id: Long,
    @SerialName("note_id") val noteId: Long,
    @SerialName("user_id") val userId: Long,
    val content: String,
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable data class RegisterReq(val username: String, val password: String, val nickname: String = "")
@Serializable data class LoginReq(val username: String, val password: String)
@Serializable data class LoginResp(val token: String)
@Serializable data class PublishReq(val title: String, val content: String, val images: List<String>)
@Serializable data class UploadResp(val url: String)
@Serializable data class CommentReq(val content: String)

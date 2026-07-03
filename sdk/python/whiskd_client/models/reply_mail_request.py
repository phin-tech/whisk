from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.mail_address import MailAddress


T = TypeVar("T", bound="ReplyMailRequest")


@_attrs_define
class ReplyMailRequest:
    """
    Attributes:
        from_ (MailAddress):
        body (str | Unset):
        payload (str | Unset):
        priority (str | Unset):
        subject (str | Unset):
        type_ (str | Unset):
    """

    from_: MailAddress
    body: str | Unset = UNSET
    payload: str | Unset = UNSET
    priority: str | Unset = UNSET
    subject: str | Unset = UNSET
    type_: str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from_ = self.from_.to_dict()

        body = self.body

        payload = self.payload

        priority = self.priority

        subject = self.subject

        type_ = self.type_

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "from": from_,
            }
        )
        if body is not UNSET:
            field_dict["body"] = body
        if payload is not UNSET:
            field_dict["payload"] = payload
        if priority is not UNSET:
            field_dict["priority"] = priority
        if subject is not UNSET:
            field_dict["subject"] = subject
        if type_ is not UNSET:
            field_dict["type"] = type_

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.mail_address import MailAddress

        d = dict(src_dict)
        from_ = MailAddress.from_dict(d.pop("from"))

        body = d.pop("body", UNSET)

        payload = d.pop("payload", UNSET)

        priority = d.pop("priority", UNSET)

        subject = d.pop("subject", UNSET)

        type_ = d.pop("type", UNSET)

        reply_mail_request = cls(
            from_=from_,
            body=body,
            payload=payload,
            priority=priority,
            subject=subject,
            type_=type_,
        )

        reply_mail_request.additional_properties = d
        return reply_mail_request

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties

from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.mail_message import MailMessage


T = TypeVar("T", bound="NextMailResponse")


@_attrs_define
class NextMailResponse:
    """
    Attributes:
        message (MailMessage | None | Unset):
        timeout (bool | Unset):
    """

    message: MailMessage | None | Unset = UNSET
    timeout: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.mail_message import MailMessage

        message: dict[str, Any] | None | Unset
        if isinstance(self.message, Unset):
            message = UNSET
        elif isinstance(self.message, MailMessage):
            message = self.message.to_dict()
        else:
            message = self.message

        timeout = self.timeout

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update({})
        if message is not UNSET:
            field_dict["message"] = message
        if timeout is not UNSET:
            field_dict["timeout"] = timeout

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.mail_message import MailMessage

        d = dict(src_dict)

        def _parse_message(data: object) -> MailMessage | None | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                message_type_1 = MailMessage.from_dict(data)

                return message_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(MailMessage | None | Unset, data)

        message = _parse_message(d.pop("message", UNSET))

        timeout = d.pop("timeout", UNSET)

        next_mail_response = cls(
            message=message,
            timeout=timeout,
        )

        next_mail_response.additional_properties = d
        return next_mail_response

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

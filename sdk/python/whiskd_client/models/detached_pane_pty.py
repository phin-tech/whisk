from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar

from attrs import define as _attrs_define
from attrs import field as _attrs_field

if TYPE_CHECKING:
    from ..models.session import Session


T = TypeVar("T", bound="DetachedPanePTY")


@_attrs_define
class DetachedPanePTY:
    """
    Attributes:
        pty_id (str):
        session (Session):
    """

    pty_id: str
    session: Session
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        pty_id = self.pty_id

        session = self.session.to_dict()

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "ptyId": pty_id,
                "session": session,
            }
        )

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.session import Session

        d = dict(src_dict)
        pty_id = d.pop("ptyId")

        session = Session.from_dict(d.pop("session"))

        detached_pane_pty = cls(
            pty_id=pty_id,
            session=session,
        )

        detached_pane_pty.additional_properties = d
        return detached_pane_pty

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

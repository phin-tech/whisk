from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.session import Session


T = TypeVar("T", bound="SplitPaneResult")


@_attrs_define
class SplitPaneResult:
    """
    Attributes:
        pane_id (str):
        session (Session):
        legacy_pty_id (str | Unset):
        pty_id (None | str | Unset):
    """

    pane_id: str
    session: Session
    legacy_pty_id: str | Unset = UNSET
    pty_id: None | str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        pane_id = self.pane_id

        session = self.session.to_dict()

        legacy_pty_id = self.legacy_pty_id

        pty_id: None | str | Unset
        if isinstance(self.pty_id, Unset):
            pty_id = UNSET
        else:
            pty_id = self.pty_id

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "paneId": pane_id,
                "session": session,
            }
        )
        if legacy_pty_id is not UNSET:
            field_dict["legacyPtyId"] = legacy_pty_id
        if pty_id is not UNSET:
            field_dict["ptyId"] = pty_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.session import Session

        d = dict(src_dict)
        pane_id = d.pop("paneId")

        session = Session.from_dict(d.pop("session"))

        legacy_pty_id = d.pop("legacyPtyId", UNSET)

        def _parse_pty_id(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        pty_id = _parse_pty_id(d.pop("ptyId", UNSET))

        split_pane_result = cls(
            pane_id=pane_id,
            session=session,
            legacy_pty_id=legacy_pty_id,
            pty_id=pty_id,
        )

        split_pane_result.additional_properties = d
        return split_pane_result

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

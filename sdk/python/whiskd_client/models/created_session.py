from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.session import Session


T = TypeVar("T", bound="CreatedSession")


@_attrs_define
class CreatedSession:
    """
    Attributes:
        pane_id (str):
        session (Session):
        window_id (str):
        main_pty_id (str | Unset):
        pty_id (None | str | Unset):
    """

    pane_id: str
    session: Session
    window_id: str
    main_pty_id: str | Unset = UNSET
    pty_id: None | str | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        pane_id = self.pane_id

        session = self.session.to_dict()

        window_id = self.window_id

        main_pty_id = self.main_pty_id

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
                "windowId": window_id,
            }
        )
        if main_pty_id is not UNSET:
            field_dict["mainPtyId"] = main_pty_id
        if pty_id is not UNSET:
            field_dict["ptyId"] = pty_id

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.session import Session

        d = dict(src_dict)
        pane_id = d.pop("paneId")

        session = Session.from_dict(d.pop("session"))

        window_id = d.pop("windowId")

        main_pty_id = d.pop("mainPtyId", UNSET)

        def _parse_pty_id(data: object) -> None | str | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            return cast(None | str | Unset, data)

        pty_id = _parse_pty_id(d.pop("ptyId", UNSET))

        created_session = cls(
            pane_id=pane_id,
            session=session,
            window_id=window_id,
            main_pty_id=main_pty_id,
            pty_id=pty_id,
        )

        created_session.additional_properties = d
        return created_session

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

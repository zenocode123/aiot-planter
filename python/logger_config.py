import logging

def setup_logger(name: str) -> logging.Logger:
    """
    設定統一格式 Logger。
    """

    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s [%(levelname)s] %(name)s: %(message)s"
    )

    return logging.getLogger(name)
    
if(_X < 140)
{
   xx = 3;
}
if(_X < 80)
{
   xx = 2;
}
if(_X < 15)
{
   xx = 1;
}
if(_Y < 660)
{
   yy = 3;
}
if(_Y < 600)
{
   yy = 2;
}
if(_Y < 540)
{
   yy = 1;
}
if(_Y < 480)
{
   yy = 0;
}
NUMBER = yy * 3 + xx;
gundisplay.gotoAndStop(NUMBER);
asdf = _parent._parent;
this.onRelease = function()
{
   asdf.player.gundisplay.gotoAndStop(NUMBER);
   updatetime = 1;
   asdf.slideback();
   asdf.slide = 1;
};
this.onRollOver = function()
{
   asdf.displaytext.text = gundisplay.id;
};
